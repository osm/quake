package message

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/state"
)

func addPlayerRecordInt16(record *[48]byte, offset int, delta int16) {
	value := int16(state.PlayerRecordUint16(record, offset)) + delta
	state.SetPlayerRecordUint16(record, offset, uint16(value))
}

func addPlayerRecordByteDelta(record *[48]byte, offset int, delta byte) {
	record[offset] = byte(int(record[offset]) + int(int8(delta)))
}

func decodePlayerInfoMaskDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	recBytes *[48]byte,
) error {
	for maskOffset, row := range playerMaskDeltaRows {
		value, err := rd.DecodeFreqByte(ft, row)
		if err != nil {
			return err
		}
		recBytes[maskOffset] ^= value
	}

	return nil
}

func decodePlayerInfoOriginDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	recBytes *[48]byte,
) error {
	originMask := recBytes[0]
	for axis, rows := range playerOriginDeltaRows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(rd, ft, uint16(originMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		addPlayerRecordInt16(recBytes, 4+axis*2, delta)
	}
	if originMask&0x40 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoFrameDelta)
		if err != nil {
			return err
		}
		addPlayerRecordByteDelta(recBytes, 10, value)
	}

	recBytes[0] &= 0xbf

	return nil
}

func decodePlayerInfoAngleDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	recBytes *[48]byte,
) error {
	angleMoveMask := recBytes[1]
	accAngle := binary.LittleEndian.Uint32(recBytes[36:40])
	for field, rows := range playerAngleMoveDeltaRows {
		lowMask := uint16(1 << uint(field*2))
		delta, err := decodeMaskedWordDelta(rd, ft, uint16(angleMoveMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		switch field {
		case 0:
			accAngle = packed.AddLow16(accAngle, delta)
			addPlayerRecordInt16(recBytes, 12, int16(uint16(accAngle)))
		case 1:
			accAngle = packed.AddHigh16(accAngle, delta)
			addPlayerRecordInt16(recBytes, 14, int16(uint16(accAngle>>16)))
		default:
			addPlayerRecordInt16(recBytes, 20+(field-2)*2, delta)
		}
	}

	binary.LittleEndian.PutUint32(recBytes[36:40], accAngle)

	return nil
}

func decodePlayerInfoStateDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	st *state.Packet,
	recBytes *[48]byte,
) error {
	stateMask := recBytes[2]

	rollDelta, err := decodeMaskedWordDelta(rd, ft, uint16(stateMask), 0x01, 0x02, playerRollDeltaRows)
	if err != nil {
		return err
	}
	addPlayerRecordInt16(recBytes, 18, rollDelta)
	if stateMask&0x04 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoButtonsXOR)
		if err != nil {
			return err
		}
		recBytes[17] ^= value
	}
	if stateMask&0x08 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoImpulseSet)
		if err != nil {
			return err
		}
		recBytes[16] = value
	}
	if stateMask&0x10 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoCommandMsecDelta)
		if err != nil {
			return err
		}
		addPlayerRecordByteDelta(recBytes, 24, value)
	}
	if stateMask&0x20 != 0 {
		remapIndex, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoModelRemapIndex)
		if err != nil {
			return err
		}
		recBytes[25] = st.ModelForRemapIndex(remapIndex)
	}
	if stateMask&0x40 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoSkinNumSet)
		if err != nil {
			return err
		}
		recBytes[26] = value
	}
	if int8(stateMask) < 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoEffectsXOR)
		if err != nil {
			return err
		}
		recBytes[27] ^= value
	}

	recBytes[2] &= 0x97

	return nil
}

func decodePlayerInfoVelocityDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	recBytes *[48]byte,
) error {
	velocityMask := recBytes[3]
	accumulatorXY := binary.LittleEndian.Uint32(recBytes[40:44])
	accumulatorZ := binary.LittleEndian.Uint32(recBytes[44:48])
	for axis, rows := range playerVelocityDeltaRows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(rd, ft, uint16(velocityMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		switch axis {
		case 0:
			accumulatorXY = packed.AddLow16(accumulatorXY, delta)
			addPlayerRecordInt16(recBytes, 28, int16(uint16(accumulatorXY)))
		case 1:
			accumulatorXY = packed.AddHigh16(accumulatorXY, delta)
			addPlayerRecordInt16(recBytes, 30, int16(uint16(accumulatorXY>>16)))
		case 2:
			accumulatorZ = packed.AddLow16(accumulatorZ, delta)
			addPlayerRecordInt16(recBytes, 32, int16(uint16(accumulatorZ)))
		}
	}

	if velocityMask&0x40 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoWeaponFrameDelta)
		if err != nil {
			return err
		}
		addPlayerRecordByteDelta(recBytes, 34, value)
	}

	recBytes[3] &= 0xbf
	binary.LittleEndian.PutUint32(recBytes[40:44], accumulatorXY)
	binary.LittleEndian.PutUint32(recBytes[44:48], accumulatorZ)

	return nil
}

func buildDecodedPlayerInfoFlags(recBytes *[48]byte, playerModelIndex byte) (uint16, byte) {
	flags, commandFlags := buildPlayerInfoFlags(recBytes, playerModelIndex)
	// Qizmo tracks buttons in its player history but does not reproduce the
	// CM_BUTTONS field when reconstructing svc_playerinfo.
	commandFlags &^= protocol.CMButtons
	if commandFlags == 0 && recBytes[24] == 0 {
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

		msec, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoMsec)
		if err != nil {
			return nil, err
		}
		record[46] = msec
		if int8(msec) < 0 {
			record[46] &= 0x7f
		} else {
			scaleStep := int(baseRecord[6] & 0xff)
			predictionScale := scaleStep
			if scaleStep != 0 {
				target := int(msec) + (packetScale - int(state.PlayerRecordByte(baseRecord, 0x2e)))
				for predictionScale < target-scaleStep {
					predictionScale += scaleStep
				}
			}
			velocityX := int16(state.PlayerRecordUint16(&record, 28))
			velocityY := int16(state.PlayerRecordUint16(&record, 30))
			velocityZ := int16(state.PlayerRecordUint16(&record, 32))
			addPlayerRecordInt16(&record, 4, packed.Scaled16(velocityX, predictionScale))
			addPlayerRecordInt16(&record, 6, packed.Scaled16(velocityY, predictionScale))
			addPlayerRecordInt16(&record, 8, packed.Scaled16(velocityZ, predictionScale))
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

	for _, freqTableAddr := range []uint32{
		freq.SVCPlayerInfoPingLo,
		freq.SVCPlayerInfoPingHi,
	} {
		b, err := rd.DecodeFreqByte(ft, freqTableAddr)
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
