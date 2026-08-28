package qizmo

import (
	"errors"
	"fmt"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
)

const maxClientSequenceDelta = 28

var errUnsupportedClientHistory = errors.New("qizmo client move history is unavailable")

type ClientEncoder struct {
	tables   *huffmanTables
	endpoint *EndpointState

	enabled   bool
	sequences sequenceTracker
	moves     clientMoveHistory
}

func NewClientEncoder(endpoint *EndpointState) (*ClientEncoder, error) {
	return newClientEncoder(freq.DefaultCompressDat, endpoint)
}

func newClientEncoder(data []byte, endpoint *EndpointState) (*ClientEncoder, error) {
	tables, err := newHuffmanTables(data)
	if err != nil {
		return nil, err
	}
	return &ClientEncoder{
		tables:   tables,
		endpoint: endpoint,
	}, nil
}

func (e *ClientEncoder) Enable() {
	e.enabled = true
}

func (e *ClientEncoder) Encode(packet []byte) ([]byte, error) {
	rawHeader, ok := readRawClientHeader(packet)
	if !ok {
		return packet, nil
	}

	sequence := rawHeader.sequence & sequenceMask
	previousSequence, hadPrevious := e.sequences.last, e.sequences.started

	nextHistory := e.moves
	operations, huffmanSupported, parseErr := parseClientOperations(packet[rawClientHeaderSize:])
	commitClientMoveRecords(&nextHistory, operations, sequence)
	// Encoding predicts from the history preceding this packet. Raw packets
	// expose all three redundant commands to Qizmo, while compressed packets
	// transmit only the commands added since the preceding sequence.
	commit := func(history clientMoveHistory) {
		e.moves = history
		e.sequences.observe(sequence)
	}

	if !e.enabled || !hadPrevious || parseErr != nil || !huffmanSupported {
		commit(nextHistory)
		return packet, nil
	}
	sequenceDelta := (sequence - previousSequence) & sequenceMask
	if sequenceDelta == 0 || sequenceDelta > maxClientSequenceDelta {
		commit(nextHistory)
		return packet, nil
	}
	// Qizmo indexes its move ring with the sequence referenced by this delta
	// before it decodes any commands. A raw packet containing only strings does
	// not populate that ring, so compressing the following packet would make
	// the original decoder reject it with an out-of-band "r" response.
	baseSequence := (sequence - sequenceDelta) & sequenceMask
	if _, ok := clientMoveRecordAt(&e.moves, baseSequence); !ok {
		commit(nextHistory)
		return packet, nil
	}

	writer := &bitWriter{}
	if err := e.writeSymbol(writer, freq.CLCSequenceDelta, byte(sequenceDelta)); err != nil {
		return nil, err
	}
	clcDelta := false
	for _, operation := range operations {
		if operation.opcode == protocol.CLCDelta && operation.payload[0] == byte(rawHeader.ack) {
			clcDelta = true
			continue
		}
		if err := e.encodeClientOperation(
			writer,
			operation,
			sequence,
			int(sequenceDelta),
		); errors.Is(err, errUnsupportedClientHistory) || errors.Is(err, errUnsupportedClientString) {
			commit(nextHistory)
			return packet, nil
		} else if err != nil {
			return nil, err
		}
	}
	if err := e.writeSymbol(writer, freq.CLCType, protocol.CLCBad); err != nil {
		return nil, err
	}
	compressedHistory := e.moves
	commitCompressedClientMoveRecords(
		&compressedHistory,
		operations,
		sequence,
		int(sequenceDelta),
	)
	commit(compressedHistory)
	header := encodeCompressedClientHeader(rawHeader, clcDelta)
	return append(header, writer.data...), nil
}

func (e *ClientEncoder) encodeClientOperation(
	writer *bitWriter,
	operation clientOperation,
	sequence uint32,
	sequenceDelta int,
) error {
	if err := e.writeSymbol(writer, freq.CLCType, operation.opcode); err != nil {
		return err
	}

	switch operation.opcode {
	case protocol.CLCMove:
		return e.encodeClientMove(writer, *operation.move, sequence, sequenceDelta)

	case protocol.CLCStringCmd:
		return e.encodeClientString(writer, operation.payload)

	case qizmoprotocol.CLCVoiceStart, qizmoprotocol.CLCVoiceStop:
		return e.encodeClientLiteralString(writer, operation.payload)

	case protocol.CLCTMove,
		qizmoprotocol.CLCPeerAssociation,
		qizmoprotocol.CLCVoiceGSM:
		writer.writeBytes(operation.payload)
		return nil

	case qizmoprotocol.CLCRequestS2CResync:
		return nil

	case protocol.CLCDelta:
		writer.writeByte(operation.payload[0])
		return nil

	case protocol.CLCUpload:
		writer.writeBytes(operation.payload[:clientUploadHeaderSize])
		previous := byte(0)
		for _, value := range operation.payload[clientUploadHeaderSize:] {
			if err := e.writeSymbol(writer, freq.CLCUploadDataByteDelta, value-previous); err != nil {
				return err
			}
			previous = value
		}
		return nil

	default:
		return fmt.Errorf("unsupported Qizmo client opcode %#x", operation.opcode)
	}
}

func (e *ClientEncoder) writeSymbol(writer *bitWriter, model uint32, symbol byte) error {
	code, err := e.tables.code(model, symbol)
	if err != nil {
		return err
	}
	writer.writeCode(code)
	return nil
}
