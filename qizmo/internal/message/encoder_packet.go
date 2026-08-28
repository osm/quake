package message

import (
	"encoding/binary"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/state"
)

const maxPacketBackReference = 30

type parsedPacket struct {
	seq        uint32
	mode       byte
	base       packetBase
	primary    primaryPlayer
	operations []serviceOperation
}

type packetBase struct {
	referenceDistance byte
	referenceSequence uint32
	predictionScale   int
	primary           state.PlayerRecord
	players           []state.PlayerRecord
	entities          []state.EntityRecord
}

type serviceOperation struct {
	opcode                byte
	data                  []byte
	players               []additionalPlayer
	entities              []state.EntityRecord
	packetEntityDelta     bool
	packetEntityDeltaBase byte
}

type operationPlan struct {
	opcode         byte
	data           []byte
	playerPlans    []additionalPlayerPlan
	packetEntities packetEntityPlan
}

type encodingContext struct {
	primary             primaryPlayer
	base                packetBase
	currentPlayers      []state.PlayerRecord
	currentEntities     []state.EntityRecord
	lastEntity          uint16
	lastCoordinates     [3]int16
	lastPingPlayerIndex byte
	lastPLPlayerIndex   byte
}

func (e *Encoder) parsePacket(
	packet []byte,
	previousSeq uint32,
	options EncodingOptions,
) (parsedPacket, bool) {
	var parsed parsedPacket
	if len(packet) < protocol.QWServerPacketHeaderSize {
		return parsed, false
	}
	seq := binary.LittleEndian.Uint32(packet)
	ack := binary.LittleEndian.Uint32(packet[protocol.QWPacketAckOffset:])
	if seq&protocol.QWSequenceReliableBit != 0 {
		if !e.qwdInput {
			return parsed, false
		}
		// Qizmo treats the high bit as a reliable-message marker while loading
		// QWD packets. The compressed stream stores only the sequence number.
		seq &= protocol.QWSequenceMask
	}
	if e.qwdInput {
		ack &= protocol.QWSequenceMask
	}
	if ack != seq && !e.qwdInput {
		return parsed, false
	}
	mode, ok := packetMode(previousSeq, seq)
	if !ok {
		return parsed, false
	}
	parsed.mode = mode
	parsed.seq = seq

	body := packet[protocol.QWServerPacketHeaderSize:]
	var heldOperations []serviceOperation
	primary, off, ok := e.parsePrimaryPlayer(body)
	var leadingPlayers []additionalPlayer
	if !ok && e.qwdInput {
		heldOperations, off = e.parseQWDLeadingOperations(body)
		var playerLength int
		primary, leadingPlayers, playerLength, ok = e.parseQWDPlayerPrefix(body[off:])
		off += playerLength
	}
	if !ok {
		return parsedPacket{}, false
	}
	parsed.primary = primary
	lastPlayer := noPlayerIndex
	if len(leadingPlayers) != 0 {
		parsed.operations = append(parsed.operations, serviceOperation{
			opcode:  protocol.SVCPlayerInfo,
			players: leadingPlayers,
		})
		lastPlayer = leadingPlayers[len(leadingPlayers)-1].player
	}
	playerOperationEnd := len(parsed.operations)

	for off < len(body) {
		operation, consumed, ok := e.parseOperationWithOptions(body[off:], options)
		if !ok {
			return parsedPacket{}, false
		}
		off += consumed

		if operation.opcode == protocol.SVCPlayerInfo {
			var groupLastPlayer byte
			operation, off, groupLastPlayer, ok = e.parsePlayerGroup(body, off, operation, lastPlayer)
			if !ok {
				return parsedPacket{}, false
			}
			if e.qwdInput {
				lastPlayer = groupLastPlayer
				// Qizmo's loader removes playerinfo records from the service
				// stream as it encounters them. Services found between two
				// player records are held until after the normal trailing block.
				heldOperations = append(heldOperations, parsed.operations[playerOperationEnd:]...)
				parsed.operations = parsed.operations[:playerOperationEnd]
				if len(parsed.operations) != 0 && parsed.operations[len(parsed.operations)-1].opcode == protocol.SVCPlayerInfo {
					last := &parsed.operations[len(parsed.operations)-1]
					last.players = append(last.players, operation.players...)
				} else {
					parsed.operations = append(parsed.operations, operation)
				}
				playerOperationEnd = len(parsed.operations)
				continue
			}
		}

		parsed.operations = append(parsed.operations, operation)
	}

	if len(parsed.operations) == 0 && mode > 1 {
		// The decoder treats this shape as an end-of-stream dropped packet.
		return parsedPacket{}, false
	}
	if e.qwdInput {
		var compatible bool
		parsed.operations, compatible = canonicalizeQWDOperations(parsed.operations, heldOperations)
		if !compatible {
			return parsedPacket{}, false
		}
	}
	if !options.DisableHistoryReference {
		parsed.base = e.selectPacketBase(seq, previousSeq)
	} else {
		parsed.base = packetBase{primary: e.state.DefaultPlayer()}
	}
	if options.PreservePacketEntityDeltas && !packetEntityBaseCompatible(parsed.operations, parsed.base) {
		return parsedPacket{}, false
	}
	return parsed, true
}

func (e *Encoder) planPacket(parsed parsedPacket) *PacketPlan {
	plan := &PacketPlan{
		encoder: e,
		seq:     parsed.seq,
		mode:    parsed.mode,
		base:    parsed.base,
		primary: e.planPrimaryPlayer(parsed.primary, parsed.base),
	}
	plan.operations = make([]operationPlan, len(parsed.operations))
	for i, operation := range parsed.operations {
		planned := &plan.operations[i]
		planned.opcode = operation.opcode
		planned.data = operation.data
		for _, player := range operation.players {
			planned.playerPlans = append(
				planned.playerPlans,
				e.planAdditionalPlayer(parsed.primary, parsed.base, player),
			)
		}
		if operation.opcode == protocol.SVCPacketEntities {
			planned.packetEntities = planPacketEntities(parsed.base.entities, operation.entities)
		}
	}
	return plan
}

func (e *Encoder) parsePlayerGroup(
	body []byte,
	offset int,
	operation serviceOperation,
	lastPlayer byte,
) (serviceOperation, int, byte, bool) {
	group := serviceOperation{opcode: protocol.SVCPlayerInfo}
	var ok bool
	if !e.qwdInput {
		lastPlayer = noPlayerIndex
	}
	for {
		if len(operation.players) != 1 {
			return serviceOperation{}, 0, 0, false
		}
		player := operation.players[0]
		if player.player >= maxPlayers || player.player-lastPlayer == 0 {
			return serviceOperation{}, 0, 0, false
		}
		group.players = append(group.players, player)
		lastPlayer = player.player

		if offset >= len(body) || body[offset] != protocol.SVCPlayerInfo {
			return group, offset, lastPlayer, true
		}
		var consumed int
		operation, consumed, ok = e.parseOperation(body[offset:])
		if !ok {
			return serviceOperation{}, 0, 0, false
		}
		offset += consumed
	}
}

func packetEntityBaseCompatible(operations []serviceOperation, base packetBase) bool {
	for _, operation := range operations {
		if operation.opcode != protocol.SVCPacketEntities {
			continue
		}
		if operation.packetEntityDelta {
			if base.referenceDistance == 0 || byte(base.referenceSequence) != operation.packetEntityDeltaBase {
				return false
			}
		} else if base.referenceDistance != 0 {
			return false
		}
	}
	return true
}

// canonicalizeQWDOperations mirrors the service-list transformations made
// by Qizmo's QWD loader before the compressor sees a packet.
func canonicalizeQWDOperations(
	operations []serviceOperation,
	held []serviceOperation,
) ([]serviceOperation, bool) {
	// The private svc_string extension is consumed by the loader itself.
	operations = discardOperations(operations, qizmoprotocol.SVCString)
	held = discardOperations(held, qizmoprotocol.SVCString)

	// Disconnect packets remain raw, as do packets without an entity snapshot.
	hasPacketEntities := false
	for _, operation := range operations {
		if operation.opcode == protocol.SVCDisconnect {
			return nil, false
		}
		if operation.opcode == protocol.SVCPacketEntities {
			hasPacketEntities = true
		}
	}
	if !hasPacketEntities {
		return nil, false
	}

	// User-info records are deferred. Held user-info records come next,
	// followed by the other services held while collecting player records.
	operations = deferOperations(operations, protocol.SVCUpdateUserInfo)
	for _, operation := range held {
		if operation.opcode == protocol.SVCUpdateUserInfo {
			operations = append(operations, operation)
		}
	}
	for _, operation := range held {
		if operation.opcode != protocol.SVCUpdateUserInfo {
			operations = append(operations, operation)
		}
	}

	// Consecutive PRINT_HIGH fragments are coalesced into one string.
	return mergeQWDPrintOperations(operations), true
}

func discardOperations(operations []serviceOperation, opcode byte) []serviceOperation {
	result := operations[:0]
	for _, operation := range operations {
		if operation.opcode != opcode {
			result = append(result, operation)
		}
	}
	return result
}

func deferOperations(operations []serviceOperation, opcode byte) []serviceOperation {
	var deferred []serviceOperation
	result := make([]serviceOperation, 0, len(operations))
	for _, operation := range operations {
		if operation.opcode == opcode {
			deferred = append(deferred, operation)
		} else {
			result = append(result, operation)
		}
	}
	return append(result, deferred...)
}

func mergeQWDPrintOperations(operations []serviceOperation) []serviceOperation {
	result := make([]serviceOperation, 0, len(operations))
	for _, operation := range operations {
		if operation.opcode == protocol.SVCPrint && len(operation.data) >= 2 &&
			operation.data[0] == protocol.PrintHigh && len(result) != 0 {
			previous := &result[len(result)-1]
			if previous.opcode == protocol.SVCPrint && len(previous.data) >= 2 &&
				previous.data[0] == operation.data[0] {
				merged := make([]byte, 0, len(previous.data)+len(operation.data)-2)
				merged = append(merged, previous.data[:len(previous.data)-1]...)
				merged = append(merged, operation.data[1:]...)
				previous.data = merged
				continue
			}
		}
		result = append(result, operation)
	}
	return result
}

func (e *Encoder) selectPacketBase(seq, previousSeq uint32) packetBase {
	base := packetBase{primary: e.state.DefaultPlayer()}
	if e.qwdInput && previousSeq&protocol.QWSequenceReliableBit != 0 {
		// A reliable-ack packet makes Qizmo suppress the immediately following
		// history reference even though the arithmetic mode still uses its
		// sequence number.
		return base
	}
	previousSeq &= protocol.QWSequenceMask
	if !e.state.AllowsHistoryReference(seq) {
		return base
	}
	if previousSeq >= seq {
		return base
	}
	referenceDistance := seq - previousSeq
	if referenceDistance > maxPacketBackReference {
		return base
	}
	entities, ok := e.state.EntitySnapshot(previousSeq)
	if !ok {
		return base
	}
	primary, found, err := e.state.FindPlayerBaseline(previousSeq)
	if err != nil {
		return base
	}
	players, ok := e.state.PlayerSnapshot(previousSeq)
	if !ok || !e.state.HasCommandScale(previousSeq) {
		return base
	}
	if !found {
		primary = e.state.DefaultPlayer()
	}
	return packetBase{
		referenceDistance: byte(referenceDistance),
		referenceSequence: previousSeq,
		predictionScale:   int(e.state.CommandScale(previousSeq)) * int(referenceDistance),
		primary:           primary,
		players:           players,
		entities:          entities,
	}
}
