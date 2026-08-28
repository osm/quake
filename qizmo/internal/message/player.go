package message

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/state"
)

func basePlayerInfoRecord(
	basePlayers []state.PlayerRecord,
	player byte,
	primaryCoordinates [3]uint16,
	playerModelIndex byte,
) state.PlayerRecord {
	// Use the prior player snapshot when available. Otherwise seed from the
	// packet's primary player position, which is how Qizmo bootstraps svc_playerinfo.
	for _, record := range basePlayers {
		if state.PlayerIndex(record) == player {
			return record
		}
	}

	var record state.PlayerRecord
	state.SetPlayerOrigin(&record, primaryCoordinates)
	state.SetPlayerModel(&record, playerModelIndex)
	state.SetPlayerIndex(&record, player)

	return record
}

func predictedPlayerOrigin(record *state.PlayerRecordBytes, scale int) (origin [3]int16) {
	for axis, field := range wire.PlayerVelocityFields {
		originOffset := wire.PlayerOriginOffset + axis*2
		origin[axis] = int16(state.PlayerRecordUint16(record, originOffset)) +
			packed.Scaled16(int16(state.PlayerRecordUint16(record, field.RecordOffset)), scale)
	}
	return origin
}

func playerVelocityAccumulatorDeltas(
	base *state.PlayerRecordBytes,
	velocity [3]int16,
) (accumulators [3]int16, deltas [3]packed.WordDelta) {
	for axis, field := range wire.PlayerVelocityFields {
		carried := int16(state.PlayerRecordUint16(base, field.RecordOffset))
		accumulators[axis] = int16(uint16(velocity[axis]) - uint16(carried))

		accumulatorOffset := wire.PlayerVelocityAccumulatorOffset + axis*2
		previous := int16(state.PlayerRecordUint16(base, accumulatorOffset))
		deltas[axis] = packed.SplitWordDelta(accumulators[axis], previous)
	}
	return accumulators, deltas
}

func playerPredictionScale(commandMsec, targetMsec, baseMsec byte, packetScale int) int {
	step := int(commandMsec)
	scale := step
	if step != 0 {
		targetScale := int(targetMsec) + packetScale - int(baseMsec)
		for scale < targetScale-step {
			scale += step
		}
	}
	return scale
}

func buildPlayerInfoFlags(record *state.PlayerRecordBytes, playerModelIndex byte) (uint16, byte) {
	flags := uint16(protocol.PFMsec)
	commandFlags := byte(0)

	for _, field := range wire.PlayerCommandFields {
		if field.RecordOffset == wire.UntrackedRecordOffset {
			continue
		}
		present := record[field.RecordOffset] != 0
		if field.Size == 2 {
			present = state.PlayerRecordUint16(record, field.RecordOffset) != 0
		}
		if present {
			commandFlags |= field.Mask
		}
	}
	if record[wire.PlayerCommandMsecOffset] != 0 || commandFlags != 0 {
		flags = protocol.PFMsec | protocol.PFCommand
	}
	for _, field := range wire.PlayerVelocityFields {
		if state.PlayerRecordUint16(record, field.RecordOffset) != 0 {
			flags |= field.Mask
		}
	}
	for _, field := range wire.PlayerByteFields {
		present := record[field.RecordOffset] != 0
		if field.Mask == protocol.PFModel {
			present = record[field.RecordOffset] != playerModelIndex
		}
		if present {
			flags |= field.Mask
		}
	}
	if record[wire.PlayerMotionMaskOffset]&wire.PlayerDead != 0 {
		flags |= protocol.PFDead
	}

	return flags, commandFlags
}

func appendPlayerInfoRecord(
	out []byte,
	player byte,
	flags uint16,
	commandFlags byte,
	record *state.PlayerRecordBytes,
) []byte {
	out = append(out, player)
	out = appendUint16LE(out, flags)
	out = append(out, record[wire.PlayerOriginOffset:wire.PlayerFrameOffset+1]...)
	out = append(out, record[wire.PlayerMsecOffset])

	if flags&protocol.PFCommand != 0 {
		out = append(out, commandFlags)
		for _, field := range wire.PlayerCommandFields {
			if field.RecordOffset != wire.UntrackedRecordOffset && commandFlags&field.Mask != 0 {
				out = append(out, record[field.RecordOffset:field.RecordOffset+field.Size]...)
			}
		}
		out = append(out, record[wire.PlayerCommandMsecOffset])
	}
	for _, field := range wire.PlayerVelocityFields {
		if flags&field.Mask != 0 {
			out = append(out, record[field.RecordOffset:field.RecordOffset+field.Size]...)
		}
	}
	for _, field := range wire.PlayerByteFields {
		if flags&field.Mask != 0 {
			out = append(out, record[field.RecordOffset])
		}
	}

	return out
}
