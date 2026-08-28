package qizmo

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
)

type clientOperation struct {
	opcode  byte
	payload []byte
	move    *clientMove
	raw     []byte // Exact bytes, present only for operations parsed from a raw packet.
}

func parseClientOperations(body []byte) ([]clientOperation, bool, error) {
	var operations []clientOperation
	huffmanSupported := true
	for len(body) != 0 {
		raw := body
		opcode := body[0]
		body = body[1:]
		operation := clientOperation{opcode: opcode}

		switch opcode {
		case protocol.CLCNOP:
			huffmanSupported = false
		case protocol.CLCMove:
			move, consumed, ok := parseClientMove(body)
			if !ok {
				return operations, false, fmt.Errorf("truncated client move")
			}
			operation.move = &move
			body = body[consumed:]
		case protocol.CLCStringCmd,
			qizmoprotocol.CLCVoiceStart,
			qizmoprotocol.CLCVoiceStop:
			end := 0
			for end < len(body) && body[end] != 0 {
				end++
			}
			if end == len(body) {
				return operations, false, fmt.Errorf("unterminated client opcode %#x string", opcode)
			}
			operation.payload = body[:end+1]
			body = body[end+1:]
		case protocol.CLCDelta:
			if len(body) < 1 {
				return operations, false, fmt.Errorf("truncated client delta")
			}
			operation.payload = body[:1]
			body = body[1:]
		case protocol.CLCTMove:
			if len(body) < clientTMovePayloadSize {
				return operations, false, fmt.Errorf("truncated client teleport move")
			}
			operation.payload = body[:clientTMovePayloadSize]
			body = body[clientTMovePayloadSize:]
		case protocol.CLCUpload:
			if len(body) < clientUploadHeaderSize {
				return operations, false, fmt.Errorf("truncated client upload header")
			}
			size, ok := clientUploadSize(body)
			if !ok {
				return operations, false, fmt.Errorf("invalid client upload size %d", size)
			}
			payloadSize := clientUploadHeaderSize + size
			if len(body) < payloadSize {
				return operations, false, fmt.Errorf("truncated client upload data")
			}
			operation.payload = body[:payloadSize]
			body = body[payloadSize:]
		case qizmoprotocol.CLCPeerAssociation:
			if len(body) < qizmoprotocol.CLCPeerAssociationPayloadSize {
				return operations, false, fmt.Errorf("truncated Qizmo peer association")
			}
			operation.payload = body[:qizmoprotocol.CLCPeerAssociationPayloadSize]
			body = body[qizmoprotocol.CLCPeerAssociationPayloadSize:]
		case qizmoprotocol.CLCVoiceRaw:
			if len(body) < qizmoprotocol.CLCVoiceRawPayloadSize {
				return operations, false, fmt.Errorf("truncated Qizmo raw voice frame")
			}
			operation.payload = body[:qizmoprotocol.CLCVoiceRawPayloadSize]
			body = body[qizmoprotocol.CLCVoiceRawPayloadSize:]
			huffmanSupported = false
		case qizmoprotocol.CLCVoiceGSM:
			if len(body) < qizmoprotocol.CLCVoiceGSMPayloadSize {
				return operations, false, fmt.Errorf("truncated Qizmo GSM voice frame")
			}
			operation.payload = body[:qizmoprotocol.CLCVoiceGSMPayloadSize]
			body = body[qizmoprotocol.CLCVoiceGSMPayloadSize:]
		case qizmoprotocol.CLCRequestS2CResync:
		default:
			return operations, false, fmt.Errorf("unsupported client opcode %#x", opcode)
		}
		operation.raw = raw[:len(raw)-len(body)]
		operations = append(operations, operation)
	}
	return operations, huffmanSupported, nil
}

func clientUploadSize(header []byte) (int, bool) {
	size := int(int16(binary.LittleEndian.Uint16(header)))
	return size, size >= 0
}

func appendClientOperation(packet []byte, operation clientOperation) []byte {
	if operation.raw != nil {
		return append(packet, operation.raw...)
	}
	packet = append(packet, operation.opcode)
	if operation.move != nil {
		return appendClientMovePayload(packet, *operation.move)
	}
	return append(packet, operation.payload...)
}

func commitClientMoveRecords(
	history *clientMoveHistory,
	operations []clientOperation,
	sequence uint32,
) {
	commitClientMoveRange(history, operations, sequence, 0, clientMoveCommandCount)
}

func commitCompressedClientMoveRecords(
	history *clientMoveHistory,
	operations []clientOperation,
	sequence uint32,
	sequenceDelta int,
) {
	firstCommand, commandCount := transmittedClientMoveRange(sequenceDelta)
	commitClientMoveRange(history, operations, sequence, firstCommand, commandCount)
}

func commitClientMoveRange(
	history *clientMoveHistory,
	operations []clientOperation,
	sequence uint32,
	firstCommand int,
	commandCount int,
) {
	for _, operation := range operations {
		if operation.move == nil {
			continue
		}
		for i := firstCommand; i < firstCommand+commandCount; i++ {
			commandSequence := clientCommandSequence(sequence, i)
			history[commandSequence&clientMoveHistoryMask] = clientMoveRecord{
				sequence: commandSequence,
				command:  operation.move.commands[i],
				lossage:  operation.move.lossage,
				valid:    true,
			}
		}
	}
}

func clientCommandSequence(packetSequence uint32, commandIndex int) uint32 {
	oldestCommandOffset := uint32(clientMoveCommandCount - 1)
	return (packetSequence - oldestCommandOffset + uint32(commandIndex)) & sequenceMask
}
