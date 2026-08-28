package message

import (
	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
)

func (e *Encoder) parseOperation(data []byte) (serviceOperation, int, bool) {
	return e.parseOperationWithOptions(data, EncodingOptions{})
}

func (e *Encoder) parseOperationWithOptions(
	data []byte,
	options EncodingOptions,
) (serviceOperation, int, bool) {
	if len(data) == 0 {
		return serviceOperation{}, 0, false
	}

	opcode := data[0]
	operation := serviceOperation{opcode: opcode}
	if size, ok := fixedPayloadSize(opcode); ok {
		consumed := 1 + size
		if consumed > len(data) {
			return serviceOperation{}, 0, false
		}
		operation.data = data[1:consumed]
		return operation, consumed, true
	}

	var consumed int
	var ok bool
	switch opcode {
	case protocol.SVCSound:
		consumed, ok = e.soundOperationLength(data)
	case protocol.SVCPrint, protocol.SVCLightStyle:
		consumed, ok = scanCStringFields(data, 2, 1)
	case protocol.SVCStuffText, protocol.SVCCenterPrint, qizmoprotocol.SVCString:
		consumed, ok = scanCStringFields(data, 1, 1)
	case protocol.SVCTempEntity:
		consumed, ok = tempEntityOperationLength(data)
	case protocol.SVCUpdateUserInfo:
		consumed, ok = scanCStringFields(data, 1+updateUserInfoStringOffset, 1)
	case protocol.SVCServerInfo:
		consumed, ok = scanCStringFields(data, 1, 2)
	case protocol.SVCSetInfo:
		consumed, ok = scanCStringFields(data, 1+setInfoKeyOffset, 2)
	case protocol.SVCNails:
		if len(data) >= 2 {
			consumed = 2 + int(data[1])*len(nailProjectileDeltaRows)
			ok = consumed <= len(data)
		}
	case protocol.SVCPlayerInfo:
		player, playerLength, parsed := e.parseAdditionalPlayer(data[1:])
		if parsed {
			operation.players = []additionalPlayer{player}
			consumed = 1 + playerLength
			ok = true
		}
	case protocol.SVCPacketEntities:
		entities, length, parsed := e.parseFullPacketEntities(data)
		if parsed {
			operation.entities = entities
			consumed = length
			ok = true
		}
	case protocol.SVCDeltaPacketEntities:
		if !options.PreservePacketEntityDeltas {
			return serviceOperation{}, 0, false
		}
		entities, base, length, parsed := e.parseDeltaPacketEntities(data)
		if parsed {
			operation.opcode = protocol.SVCPacketEntities
			operation.entities = entities
			operation.packetEntityDelta = true
			operation.packetEntityDeltaBase = base
			consumed = length
			ok = true
		}
	default:
		return serviceOperation{}, 0, false
	}
	if !ok {
		return serviceOperation{}, 0, false
	}
	if len(operation.players) == 0 && len(operation.entities) == 0 {
		operation.data = data[1:consumed]
	}
	return operation, consumed, true
}

func fixedPayloadSize(opcode byte) (int, bool) {
	if codec, ok := fixedOperationCodecs[opcode]; ok {
		return codec.size, true
	}
	switch opcode {
	case protocol.SVCUpdatePing:
		return 3, true
	case protocol.SVCDamage:
		return 2 + coordinateTripletSize, true
	case protocol.SVCMuzzleFlash, protocol.SVCUpdatePL:
		return 2, true
	case qizmoprotocol.SVCVoice:
		return qizmoprotocol.SVCVoicePayloadSize, true
	default:
		return 0, false
	}
}

func scanCStringFields(data []byte, off, count int) (int, bool) {
	if off > len(data) {
		return 0, false
	}
	var ok bool
	for range count {
		off, ok = endCString(data, off)
		if !ok {
			return 0, false
		}
	}
	return off, true
}

// parseQWDLeadingOperations consumes the ordinary services Qizmo accepts
// before the packet's primary svc_playerinfo record. Player and entity records
// mark the end of this prefix and are handled by their state-aware parsers.
func (e *Encoder) parseQWDLeadingOperations(body []byte) ([]serviceOperation, int) {
	var operations []serviceOperation
	off := 0
	for off < len(body) {
		switch body[off] {
		case protocol.SVCPlayerInfo, protocol.SVCPacketEntities:
			return operations, off
		}

		operation, consumed, ok := e.parseOperation(body[off:])
		if !ok {
			return operations, off
		}
		operations = append(operations, operation)
		off += consumed
	}
	return operations, off
}

func (e *Encoder) parseQWDPlayerPrefix(
	body []byte,
) (primaryPlayer, []additionalPlayer, int, bool) {
	var primary primaryPlayer
	var players []additionalPlayer
	foundPrimary := false
	off := 0
	for off < len(body) && body[off] == protocol.SVCPlayerInfo {
		if off+1 >= len(body) {
			return primaryPlayer{}, nil, 0, false
		}

		var consumed int
		var ok bool
		if body[off+1] == e.state.PlayerIndex {
			if foundPrimary {
				return primaryPlayer{}, nil, 0, false
			}
			primary, consumed, ok = e.parsePrimaryPlayer(body[off:])
			foundPrimary = ok
		} else {
			var player additionalPlayer
			player, consumed, ok = e.parseAdditionalPlayer(body[off+1:])
			consumed++ // Include the svc_playerinfo opcode.
			if ok {
				players = append(players, player)
			}
		}
		if !ok || consumed <= 1 {
			return primaryPlayer{}, nil, 0, false
		}
		off += consumed
	}
	return primary, players, off, foundPrimary
}
