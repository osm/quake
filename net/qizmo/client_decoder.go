package qizmo

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
)

type ClientDecoder struct {
	tables    *huffmanTables
	endpoint  *EndpointState
	sequences qizmoprotocol.LinkSequences
	moves     clientMoveHistory
}

func NewClientDecoder(endpoint *EndpointState) (*ClientDecoder, error) {
	return newClientDecoder(freq.DefaultCompressDat, endpoint)
}

func newClientDecoder(data []byte, endpoint *EndpointState) (*ClientDecoder, error) {
	tables, err := newHuffmanTables(data)
	if err != nil {
		return nil, err
	}
	return &ClientDecoder{
		tables:   tables,
		endpoint: endpoint,
	}, nil
}

func (d *ClientDecoder) Decode(packet []byte) ([]byte, error) {
	if !qizmoprotocol.IsCompressedLinkPacket(packet) {
		// Unknown raw extensions must remain valid passthrough packets.
		_, _ = d.observeRawPacket(packet)
		return packet, nil
	}
	decoded, _, err := d.decodeCompressedPacket(packet)
	return decoded, err
}

func (d *ClientDecoder) decodeCompressedPacket(packet []byte) ([]byte, []clientOperation, error) {
	if len(packet) < compressedClientHeaderSize {
		return nil, nil, fmt.Errorf("compressed Qizmo client packet is too short: %d", len(packet))
	}

	header, body, err := d.sequences.DecodeHeader(packet)
	if err != nil {
		return nil, nil, err
	}
	if len(body) < qPortSize {
		return nil, nil, fmt.Errorf("compressed Qizmo client packet has no qport")
	}

	reader := &bitReader{data: body[qPortSize:]}
	sequenceDelta, err := d.readSymbol(reader, freq.CLCSequenceDelta)
	if err != nil {
		return nil, nil, fmt.Errorf("decode client sequence delta: %w", err)
	}
	if sequenceDelta == 0 || sequenceDelta > maxClientSequenceDelta {
		return nil, nil, fmt.Errorf("invalid Qizmo client sequence delta %d", sequenceDelta)
	}

	sequence := header.Sequence & sequenceMask
	operations, err := d.decodeClientOperations(reader, sequence, int(sequenceDelta))
	if err != nil {
		return nil, nil, err
	}
	if header.CLCDelta {
		// Qizmo folds a clc_delta matching the acknowledgement into its header.
		operations = append(operations, clientOperation{
			opcode:  protocol.CLCDelta,
			payload: []byte{byte(header.Ack)},
		})
	}

	decoded := binary.LittleEndian.AppendUint32(nil, header.Sequence)
	decoded = binary.LittleEndian.AppendUint32(decoded, header.Ack)
	decoded = append(decoded, body[:qPortSize]...)
	for _, operation := range operations {
		decoded = appendClientOperation(decoded, operation)
	}
	commitCompressedClientMoveRecords(&d.moves, operations, sequence, int(sequenceDelta))
	d.observeS2CResyncRequest(operations)
	return decoded, operations, nil
}

func (d *ClientDecoder) observeRawPacket(packet []byte) ([]clientOperation, error) {
	header, ok := readRawClientHeader(packet)
	if !ok {
		return nil, nil
	}
	d.sequences.Observe(header.sequence, header.ack)

	sequence := header.sequence & sequenceMask
	operations, _, parseErr := parseClientOperations(packet[rawClientHeaderSize:])
	commitClientMoveRecords(&d.moves, operations, sequence)
	if parseErr != nil {
		return operations, parseErr
	}
	d.observeS2CResyncRequest(operations)
	return operations, nil
}

func (d *ClientDecoder) decodeClientOperations(
	reader *bitReader,
	sequence uint32,
	sequenceDelta int,
) ([]clientOperation, error) {
	var operations []clientOperation
	for {
		opcode, err := d.readSymbol(reader, freq.CLCType)
		if err != nil {
			return nil, fmt.Errorf("decode client opcode: %w", err)
		}
		if opcode == protocol.CLCBad {
			return operations, nil
		}

		operation := clientOperation{opcode: opcode}
		switch opcode {
		case protocol.CLCMove:
			move, err := d.decodeClientMove(reader, sequence, sequenceDelta)
			if err != nil {
				return nil, err
			}
			operation.move = &move

		case protocol.CLCStringCmd:
			operation.payload, err = d.decodeClientString(reader)
			if err != nil {
				return nil, fmt.Errorf("decode client string: %w", err)
			}

		case qizmoprotocol.CLCVoiceStart, qizmoprotocol.CLCVoiceStop:
			operation.payload, err = d.decodeClientLiteralString(reader)
			if err != nil {
				return nil, fmt.Errorf("decode Qizmo voice stream control: %w", err)
			}

		case qizmoprotocol.CLCPeerAssociation:
			operation.payload, err = reader.readBytes(qizmoprotocol.CLCPeerAssociationPayloadSize)
			if err != nil {
				return nil, fmt.Errorf("decode Qizmo peer association: %w", err)
			}

		case qizmoprotocol.CLCVoiceGSM:
			operation.payload, err = reader.readBytes(qizmoprotocol.CLCVoiceGSMPayloadSize)
			if err != nil {
				return nil, fmt.Errorf("decode Qizmo GSM voice frame: %w", err)
			}

		case qizmoprotocol.CLCRequestS2CResync:

		case protocol.CLCDelta:
			value, err := reader.readByte()
			if err != nil {
				return nil, fmt.Errorf("decode client delta: %w", err)
			}
			operation.payload = []byte{value}

		case protocol.CLCTMove:
			operation.payload, err = reader.readBytes(clientTMovePayloadSize)
			if err != nil {
				return nil, fmt.Errorf("decode client teleport move: %w", err)
			}

		case protocol.CLCUpload:
			header, err := reader.readBytes(clientUploadHeaderSize)
			if err != nil {
				return nil, fmt.Errorf("decode client upload header: %w", err)
			}
			size, ok := clientUploadSize(header)
			if !ok {
				return nil, fmt.Errorf("invalid Qizmo client upload size %d", size)
			}
			operation.payload = header
			previous := byte(0)
			for range size {
				delta, err := d.readSymbol(reader, freq.CLCUploadDataByteDelta)
				if err != nil {
					return nil, fmt.Errorf("decode client upload data: %w", err)
				}
				previous += delta
				operation.payload = append(operation.payload, previous)
			}

		default:
			return nil, fmt.Errorf("unsupported Qizmo client opcode %#x", opcode)
		}
		operations = append(operations, operation)
	}
}

func (d *ClientDecoder) readSymbol(reader *bitReader, model uint32) (byte, error) {
	return d.tables.readSymbol(reader, model)
}
