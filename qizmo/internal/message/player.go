package message

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/internal/wire"
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
	for _, rec := range basePlayers {
		if state.PlayerIndex(rec) == player {
			return rec
		}
	}

	var rec state.PlayerRecord
	rec[1] = uint32(primaryCoordinates[0]) | uint32(primaryCoordinates[1])<<16
	rec[2] = uint32(primaryCoordinates[2])
	state.SetPlayerModel(&rec, playerModelIndex)
	state.SetPlayerIndex(&rec, player)

	return rec
}

func buildPlayerInfoFlags(recBytes *[48]byte, playerModelIndex byte) (uint16, byte) {
	flags := uint16(protocol.PFMsec)
	commandFlags := byte(0)

	for _, field := range wire.PlayerCommandFields {
		if field.RecordOffset == wire.UntrackedRecordOffset {
			continue
		}
		present := recBytes[field.RecordOffset] != 0
		if field.Size == 2 {
			present = state.PlayerRecordUint16(recBytes, field.RecordOffset) != 0
		}
		if present {
			commandFlags |= field.Mask
		}
	}
	if recBytes[24] != 0 || commandFlags != 0 {
		flags = protocol.PFMsec | protocol.PFCommand
	}
	for _, field := range wire.PlayerVelocityFields {
		if state.PlayerRecordUint16(recBytes, field.RecordOffset) != 0 {
			flags |= field.Mask
		}
	}
	for _, field := range wire.PlayerByteFields {
		present := recBytes[field.RecordOffset] != 0
		if field.Mask == protocol.PFModel {
			present = recBytes[field.RecordOffset] != playerModelIndex
		}
		if present {
			flags |= field.Mask
		}
	}
	if int8(recBytes[3]) < 0 {
		flags |= protocol.PFDead
	}

	return flags, commandFlags
}

func appendPlayerInfoRecord(
	out []byte,
	player byte,
	flags uint16,
	commandFlags byte,
	recBytes *[48]byte,
) []byte {
	out = append(out, player)
	out = appendUint16LE(out, flags)
	out = append(
		out,
		recBytes[4], recBytes[5], recBytes[6], recBytes[7],
		recBytes[8], recBytes[9], recBytes[10], recBytes[46],
	)

	if flags&protocol.PFCommand != 0 {
		out = append(out, commandFlags)
		for _, field := range wire.PlayerCommandFields {
			if field.RecordOffset != wire.UntrackedRecordOffset && commandFlags&field.Mask != 0 {
				out = append(out, recBytes[field.RecordOffset:field.RecordOffset+field.Size]...)
			}
		}
		out = append(out, recBytes[24])
	}
	for _, field := range wire.PlayerVelocityFields {
		if flags&field.Mask != 0 {
			out = append(out, recBytes[field.RecordOffset:field.RecordOffset+field.Size]...)
		}
	}
	for _, field := range wire.PlayerByteFields {
		if flags&field.Mask != 0 {
			out = append(out, recBytes[field.RecordOffset])
		}
	}

	return out
}
