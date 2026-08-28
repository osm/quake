package message

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/state"
)

func addPlayerRecordInt16(record *state.PlayerRecordBytes, offset int, delta int16) {
	value := int16(state.PlayerRecordUint16(record, offset)) + delta
	state.SetPlayerRecordUint16(record, offset, uint16(value))
}

func addPlayerRecordByteDelta(record *state.PlayerRecordBytes, offset int, delta byte) {
	record[offset] = byte(int(record[offset]) + int(int8(delta)))
}

func decodePlayerInfoMaskDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	record *state.PlayerRecordBytes,
) error {
	for maskOffset, row := range playerMaskDeltaRows {
		value, err := rd.DecodeFreqByte(ft, row)
		if err != nil {
			return err
		}
		record[maskOffset] ^= value
	}

	return nil
}

func decodePlayerInfoOriginDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	record *state.PlayerRecordBytes,
) error {
	originMask := record[wire.PlayerOriginMaskOffset]
	for axis, rows := range playerOriginDeltaRows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(rd, ft, uint16(originMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		addPlayerRecordInt16(record, wire.PlayerOriginOffset+axis*2, delta)
	}
	if originMask&wire.PlayerFrameDelta != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoFrameDelta)
		if err != nil {
			return err
		}
		addPlayerRecordByteDelta(record, wire.PlayerFrameOffset, value)
	}

	record[wire.PlayerOriginMaskOffset] &= wire.PlayerOriginHistoryMask

	return nil
}

func decodePlayerInfoAngleDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	record *state.PlayerRecordBytes,
) error {
	angleMoveMask := record[wire.PlayerAngleMoveMaskOffset]
	accumulator := binary.LittleEndian.Uint32(record[wire.PlayerAngleAccumulatorOffset:])
	for field, rows := range playerAngleMoveDeltaRows {
		lowMask := uint16(1 << uint(field*2))
		delta, err := decodeMaskedWordDelta(rd, ft, uint16(angleMoveMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		switch field {
		case 0:
			accumulator = packed.AddLow16(accumulator, delta)
			addPlayerRecordInt16(record, wire.PlayerAngleOffset, int16(uint16(accumulator)))
		case 1:
			accumulator = packed.AddHigh16(accumulator, delta)
			addPlayerRecordInt16(record, wire.PlayerAngleOffset+2, int16(uint16(accumulator>>16)))
		default:
			addPlayerRecordInt16(record, wire.PlayerMoveOffset+(field-2)*2, delta)
		}
	}

	binary.LittleEndian.PutUint32(record[wire.PlayerAngleAccumulatorOffset:], accumulator)

	return nil
}

func decodePlayerInfoStateDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	st *state.Packet,
	record *state.PlayerRecordBytes,
) error {
	stateMask := record[wire.PlayerStateMaskOffset]

	rollDelta, err := decodeMaskedWordDelta(
		rd,
		ft,
		uint16(stateMask),
		wire.PlayerRollDeltaLo,
		wire.PlayerRollDeltaHi,
		playerRollDeltaRows,
	)
	if err != nil {
		return err
	}
	addPlayerRecordInt16(record, wire.PlayerRollOffset, rollDelta)
	if stateMask&wire.PlayerButtonsXOR != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoButtonsXOR)
		if err != nil {
			return err
		}
		record[wire.PlayerButtonsOffset] ^= value
	}
	if stateMask&wire.PlayerImpulseSet != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoImpulseSet)
		if err != nil {
			return err
		}
		record[wire.PlayerImpulseOffset] = value
	}
	if stateMask&wire.PlayerCommandMsecDelta != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoCommandMsecDelta)
		if err != nil {
			return err
		}
		addPlayerRecordByteDelta(record, wire.PlayerCommandMsecOffset, value)
	}
	if stateMask&wire.PlayerModelRemap != 0 {
		remapIndex, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoModelRemapIndex)
		if err != nil {
			return err
		}
		record[wire.PlayerModelOffset] = st.ModelForRemapIndex(remapIndex)
	}
	if stateMask&wire.PlayerSkinNumSet != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoSkinNumSet)
		if err != nil {
			return err
		}
		record[wire.PlayerSkinNumOffset] = value
	}
	if stateMask&wire.PlayerEffectsXOR != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoEffectsXOR)
		if err != nil {
			return err
		}
		record[wire.PlayerEffectsOffset] ^= value
	}

	record[wire.PlayerStateMaskOffset] &= wire.PlayerStateHistoryMask

	return nil
}

func decodePlayerInfoVelocityDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	record *state.PlayerRecordBytes,
) error {
	velocityMask := record[wire.PlayerMotionMaskOffset]
	accumulatorXY := binary.LittleEndian.Uint32(record[wire.PlayerVelocityAccumulatorOffset:])
	accumulatorZ := binary.LittleEndian.Uint32(record[wire.PlayerVelocityAccumulatorOffset+4:])
	for axis, rows := range playerVelocityDeltaRows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(rd, ft, uint16(velocityMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		switch axis {
		case 0:
			accumulatorXY = packed.AddLow16(accumulatorXY, delta)
			addPlayerRecordInt16(record, wire.PlayerVelocityOffset, int16(uint16(accumulatorXY)))
		case 1:
			accumulatorXY = packed.AddHigh16(accumulatorXY, delta)
			addPlayerRecordInt16(record, wire.PlayerVelocityOffset+2, int16(uint16(accumulatorXY>>16)))
		case 2:
			accumulatorZ = packed.AddLow16(accumulatorZ, delta)
			addPlayerRecordInt16(record, wire.PlayerVelocityOffset+4, int16(uint16(accumulatorZ)))
		}
	}

	if velocityMask&wire.PlayerWeaponFrameDelta != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoWeaponFrameDelta)
		if err != nil {
			return err
		}
		addPlayerRecordByteDelta(record, wire.PlayerWeaponFrameOffset, value)
	}

	record[wire.PlayerMotionMaskOffset] &= wire.PlayerMotionHistoryMask
	binary.LittleEndian.PutUint32(record[wire.PlayerVelocityAccumulatorOffset:], accumulatorXY)
	binary.LittleEndian.PutUint32(record[wire.PlayerVelocityAccumulatorOffset+4:], accumulatorZ)

	return nil
}

func buildDecodedPlayerInfoFlags(record *state.PlayerRecordBytes, playerModelIndex byte) (uint16, byte) {
	flags, commandFlags := buildPlayerInfoFlags(record, playerModelIndex)
	// Qizmo tracks buttons in its player history but does not reproduce the
	// CM_BUTTONS field when reconstructing svc_playerinfo.
	commandFlags &^= protocol.CMButtons
	if commandFlags == 0 && record[wire.PlayerCommandMsecOffset] == 0 {
		flags &^= protocol.PFCommand
	}
	return flags, commandFlags
}

func (d *packetDecoder) decodePlayerSlotDelta(lastPlayerIndex *byte) (byte, error) {
	delta, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerIndexDelta)
	if err != nil {
		return 0, err
	}

	*lastPlayerIndex += delta

	return *lastPlayerIndex, nil
}

func (d *packetDecoder) decodeSVCPlayerInfoDeltas(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	basePlayers := d.basePlayers
	packetScale := d.packetScale
	lastPlayerIndex := &d.lastPlayerIndex

	firstPlayerInfo := true
	for {
		delta, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoNumDelta)
		if err != nil {
			return nil, err
		}
		if delta == 0 {
			return out, nil
		}

		player := *lastPlayerIndex + delta
		*lastPlayerIndex = player
		if player >= maxPlayers {
			return nil, fmt.Errorf(
				"%w: invalid svc_playerinfo player %d",
				errDroppedPacket,
				player,
			)
		}

		baseRecord := basePlayerInfoRecord(
			basePlayers,
			player,
			d.primaryCoordinates,
			st.PlayerModelIndex,
		)

		record := state.PlayerRecordBytesLE(baseRecord)

		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoMsec)
		if err != nil {
			return nil, err
		}
		record[wire.PlayerMsecOffset] = b
		if b&wire.PlayerMsecShortcut != 0 {
			record[wire.PlayerMsecOffset] &^= wire.PlayerMsecShortcut
		} else {
			scale := playerPredictionScale(
				state.PlayerRecordByte(baseRecord, wire.PlayerCommandMsecOffset),
				b,
				state.PlayerRecordByte(baseRecord, wire.PlayerMsecOffset),
				packetScale,
			)
			predictedOrigin := predictedPlayerOrigin(&record, scale)
			for axis, value := range predictedOrigin {
				state.SetPlayerRecordUint16(&record, wire.PlayerOriginOffset+axis*2, uint16(value))
			}
			if err := decodePlayerInfoMaskDeltas(rd, ft, &record); err != nil {
				return nil, err
			}
			if err := decodePlayerInfoOriginDeltas(rd, ft, &record); err != nil {
				return nil, err
			}
			if err := decodePlayerInfoAngleDeltas(rd, ft, &record); err != nil {
				return nil, err
			}
			if err := decodePlayerInfoStateDeltas(rd, ft, st, &record); err != nil {
				return nil, err
			}
			if err := decodePlayerInfoVelocityDeltas(rd, ft, &record); err != nil {
				return nil, err
			}
		}

		flags, commandFlags := buildDecodedPlayerInfoFlags(&record, st.PlayerModelIndex)

		if !firstPlayerInfo {
			out = append(out, protocol.SVCPlayerInfo)
		}
		firstPlayerInfo = false
		out = appendPlayerInfoRecord(out, player, flags, commandFlags, &record)

		st.CurrentPlayers = append(
			st.CurrentPlayers,
			state.PlayerRecordFromBytesLE(record),
		)
	}
}

func (d *packetDecoder) decodeSVCUpdatePing(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft

	playerIndex, err := d.decodePlayerSlotDelta(&d.lastPingPlayerIndex)
	if err != nil {
		return nil, err
	}
	out = append(out, playerIndex)

	for _, row := range []uint32{
		freq.SVCPlayerInfoPingLo,
		freq.SVCPlayerInfoPingHi,
	} {
		b, err := rd.DecodeFreqByte(ft, row)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (d *packetDecoder) decodeSVCUpdatePL(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft

	playerIndex, err := d.decodePlayerSlotDelta(&d.lastPLPlayerIndex)
	if err != nil {
		return nil, err
	}
	out = append(out, playerIndex)

	b, err := rd.DecodeFreqByte(ft, freq.SVCUpdatePLPacketLossByte)
	if err != nil {
		return nil, err
	}
	return append(out, b), nil
}
