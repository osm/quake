package compressed

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/qwz/freq"
	"github.com/osm/quake/demo/qwz/packed"
	"github.com/osm/quake/demo/qwz/rangedec"
	"github.com/osm/quake/demo/qwz/state"
)

func addPlayerRecordI16(b *[48]byte, off int, delta int16) {
	value := int16(state.GetU16LE(b, off)) + delta
	state.SetU16LE(b, off, uint16(value))
}

func addPlayerRecordByteDelta(b *[48]byte, off int, delta byte) {
	b[off] = byte(int(b[off]) + int(int8(delta)))
}

func decodePlayerInfoMaskDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	recBytes *[48]byte,
) error {
	for _, spec := range []struct {
		maskOff       int
		freqTableAddr uint32
	}{
		{0, freq.SVCPlayerInfoOriginMaskDelta},
		{1, freq.SVCPlayerInfoAngleMoveMaskDelta},
		{2, freq.SVCPlayerInfoStateMaskDelta},
		{3, freq.SVCPlayerInfoVelocityMaskDelta},
	} {
		value, err := rd.DecodeFreqByte(ft, spec.freqTableAddr)
		if err != nil {
			return err
		}
		recBytes[spec.maskOff] ^= value
	}

	return nil
}

func decodePlayerInfoOriginDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	recBytes *[48]byte,
) error {
	originMask := recBytes[0]

	if originMask&0x01 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginXLoDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 4, int16(int8(value)))
	}
	if originMask&0x02 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginXHiDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 4, int16(uint16(value)<<8))
	}
	if originMask&0x04 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginYLoDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 6, int16(int8(value)))
	}
	if originMask&0x08 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginYHiDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 6, int16(uint16(value)<<8))
	}
	if originMask&0x10 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginZLoDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 8, int16(int8(value)))
	}
	if originMask&0x20 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginZHiDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 8, int16(uint16(value)<<8))
	}
	if originMask&0x40 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginZByte2Delta)
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

	if angleMoveMask&0x01 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoPitchLoDelta)
		if err != nil {
			return err
		}
		accAngle = packed.AddLow16(accAngle, int16(int8(value)))
	}
	if angleMoveMask&0x02 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoPitchHiDelta)
		if err != nil {
			return err
		}
		accAngle = packed.AddLow16(accAngle, int16(uint16(value)<<8))
	}
	addPlayerRecordI16(recBytes, 12, int16(uint16(accAngle&0xffff)))

	if angleMoveMask&0x04 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoYawLoDelta)
		if err != nil {
			return err
		}
		accAngle = packed.AddHigh16(accAngle, int16(int8(value)))
	}
	if angleMoveMask&0x08 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoYawHiDelta)
		if err != nil {
			return err
		}
		accAngle = packed.AddHigh16(accAngle, int16(uint16(value)<<8))
	}
	addPlayerRecordI16(recBytes, 14, int16(uint16(accAngle>>16)))

	if angleMoveMask&0x10 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoForwardLoDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 20, int16(int8(value)))
	}
	if angleMoveMask&0x20 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoForwardHiDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 20, int16(uint16(value)<<8))
	}
	if angleMoveMask&0x40 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoSideLoDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 22, int16(int8(value)))
	}
	if int8(angleMoveMask) < 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoSideHiDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 22, int16(uint16(value)<<8))
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

	if stateMask&0x01 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoRollLoDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 18, int16(int8(value)))
	}
	if stateMask&0x02 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoRollHiDelta)
		if err != nil {
			return err
		}
		addPlayerRecordI16(recBytes, 18, int16(uint16(value)<<8))
	}
	if stateMask&0x04 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoImpulseXor)
		if err != nil {
			return err
		}
		recBytes[17] ^= value
	}
	if stateMask&0x08 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoButtonsSet)
		if err != nil {
			return err
		}
		recBytes[16] = value
	}
	if stateMask&0x10 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoFrameDelta)
		if err != nil {
			return err
		}
		addPlayerRecordByteDelta(recBytes, 24, value)
	}
	if stateMask&0x20 != 0 {
		idx, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoFreqRemapIndex)
		if err != nil {
			return err
		}
		recBytes[25] = st.ModelRemapByte(idx)
	}
	if stateMask&0x40 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoEffectsSet)
		if err != nil {
			return err
		}
		recBytes[26] = value
	}
	if int8(stateMask) < 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoSkinXor)
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
	accVelXY := binary.LittleEndian.Uint32(recBytes[40:44])
	accVelZ := binary.LittleEndian.Uint32(recBytes[44:48])

	if velocityMask&0x01 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelXLoDelta)
		if err != nil {
			return err
		}
		accVelXY = packed.AddLow16(accVelXY, int16(int8(value)))
	}
	if velocityMask&0x02 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelXHiDelta)
		if err != nil {
			return err
		}
		accVelXY = packed.AddLow16(accVelXY, int16(uint16(value)<<8))
	}
	addPlayerRecordI16(recBytes, 28, int16(uint16(accVelXY&0xffff)))

	if velocityMask&0x04 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelYLoDelta)
		if err != nil {
			return err
		}
		accVelXY = packed.AddHigh16(accVelXY, int16(int8(value)))
	}
	if velocityMask&0x08 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelYHiDelta)
		if err != nil {
			return err
		}
		accVelXY = packed.AddHigh16(accVelXY, int16(uint16(value)<<8))
	}
	addPlayerRecordI16(recBytes, 30, int16(uint16(accVelXY>>16)))

	if velocityMask&0x10 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelZLoDelta)
		if err != nil {
			return err
		}
		accVelZ = packed.AddLow16(accVelZ, int16(int8(value)))
	}
	if velocityMask&0x20 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelZHiDelta)
		if err != nil {
			return err
		}
		accVelZ = packed.AddLow16(accVelZ, int16(uint16(value)<<8))
	}
	addPlayerRecordI16(recBytes, 32, int16(uint16(accVelZ&0xffff)))

	if velocityMask&0x40 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoWeaponFrameDelta)
		if err != nil {
			return err
		}
		addPlayerRecordByteDelta(recBytes, 34, value)
	}

	recBytes[3] &= 0xbf
	binary.LittleEndian.PutUint32(recBytes[40:44], accVelXY)
	binary.LittleEndian.PutUint32(recBytes[44:48], accVelZ)

	return nil
}

func basePlayerInfoRecord(
	basePlayers []state.PlayerRecord,
	player byte,
	primaryPlayerPosXY uint32,
	primaryPlayerPosZ uint32,
	playerFreqIndex byte,
) state.PlayerRecord {
	// Use the prior player snapshot when available. Otherwise seed from the
	// packet's primary player position, which is how Qizmo bootstraps 0x2a.
	for _, rec := range basePlayers {
		if state.PlayerRecordByte(rec, 0x0b) == player {
			return rec
		}
	}

	var rec state.PlayerRecord
	rec[1] = primaryPlayerPosXY
	rec[2] = uint32(uint16(primaryPlayerPosZ))
	state.SetPlayerRecordByte(&rec, 0x19, playerFreqIndex)
	state.SetPlayerRecordByte(&rec, 0x0b, player)

	return rec
}

func buildPlayerInfoFlags(recBytes *[48]byte, playerFreqIndex byte) (uint16, byte) {
	// Reconstruct the wire svc_playerinfo flags from the packed snapshot state.
	flags := uint16(0x01)
	extraFlags := byte(0)

	if state.GetU16LE(recBytes, 12) != 0 {
		extraFlags |= 0x80
	}
	if state.GetU16LE(recBytes, 14) != 0 {
		extraFlags |= 0x01
	}
	if state.GetU16LE(recBytes, 20) != 0 {
		extraFlags |= 0x04
	}
	if state.GetU16LE(recBytes, 22) != 0 {
		extraFlags |= 0x08
	}
	if state.GetU16LE(recBytes, 18) != 0 {
		extraFlags |= 0x10
	}
	if recBytes[16] != 0 {
		extraFlags |= 0x40
	}
	if recBytes[24] != 0 || extraFlags != 0 {
		flags = 0x03
	}
	if recBytes[25] != playerFreqIndex {
		flags |= 0x20
	}
	if recBytes[26] != 0 {
		flags |= 0x40
	}
	if recBytes[27] != 0 {
		flags |= 0x80
	}
	if state.GetU16LE(recBytes, 28) != 0 {
		flags |= 0x04
	}
	if state.GetU16LE(recBytes, 30) != 0 {
		flags |= 0x08
	}
	if state.GetU16LE(recBytes, 32) != 0 {
		flags |= 0x10
	}
	if recBytes[34] != 0 {
		flags |= 0x100
	}
	if int8(recBytes[3]) < 0 {
		flags |= 0x200
	}

	return flags, extraFlags
}

func appendPlayerInfoRecord(
	out []byte,
	player byte,
	flags uint16,
	extraFlags byte,
	recBytes *[48]byte,
) []byte {
	out = append(out, player)
	out = appendUint16LE(out, flags)
	out = append(
		out,
		recBytes[4], recBytes[5], recBytes[6], recBytes[7],
		recBytes[8], recBytes[9], recBytes[10], recBytes[46],
	)

	if flags&0x02 != 0 {
		out = append(out, extraFlags)
		if extraFlags&0x01 != 0 {
			out = append(out, recBytes[14], recBytes[15])
		}
		if extraFlags&0x80 != 0 {
			out = append(out, recBytes[12], recBytes[13])
		}
		if extraFlags&0x04 != 0 {
			out = append(out, recBytes[20], recBytes[21])
		}
		if extraFlags&0x08 != 0 {
			out = append(out, recBytes[22], recBytes[23])
		}
		if extraFlags&0x10 != 0 {
			out = append(out, recBytes[18], recBytes[19])
		}
		if extraFlags&0x40 != 0 {
			out = append(out, recBytes[16])
		}
		out = append(out, recBytes[24])
	}
	if flags&0x04 != 0 {
		out = append(out, recBytes[28], recBytes[29])
	}
	if flags&0x08 != 0 {
		out = append(out, recBytes[30], recBytes[31])
	}
	if flags&0x10 != 0 {
		out = append(out, recBytes[32], recBytes[33])
	}
	if flags&0x20 != 0 {
		out = append(out, recBytes[25])
	}
	if flags&0x40 != 0 {
		out = append(out, recBytes[26])
	}
	if flags&0x80 != 0 {
		out = append(out, recBytes[27])
	}
	if flags&0x100 != 0 {
		out = append(out, recBytes[34])
	}

	return out
}

func (d *decoder) decodePlayerSlotDelta(lastPlayerIndex *byte) (byte, error) {
	delta, err := d.rd.DecodeFreqByte(d.ft, freq.PlayerInfoSlot2)
	if err != nil {
		return 0, err
	}

	*lastPlayerIndex += delta

	return *lastPlayerIndex, nil
}

func (d *decoder) decodeSVCPlayerInfo() ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	out := make([]byte, 0, 64)

	out = append(out, st.PlayerIndex)

	back, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoBackref)
	if err != nil {
		return nil, err
	}
	refSeq := st.SeqNo() - uint32(back)
	scale := int(st.Scale(refSeq)) * int(back)

	var rec state.PlayerRecord
	if back == 0 {
		rec = st.DefaultRec()
		st.PacketHasBase = false
	} else {
		st.PacketBaseSeq = refSeq
		st.PacketHasBase = true
		base, ok, err := st.FindBaseline(refSeq)
		if err != nil {
			return nil, err
		}
		if ok {
			rec = base
		} else {
			rec = st.DefaultRec()
		}
	}

	b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginMaskXor)
	if err != nil {
		return nil, err
	}
	state.SetPlayerOriginMask(&rec, state.PlayerOriginMask(rec)^b)

	b, err = rd.DecodeFreqByte(ft, freq.SVCPlayerInfoStateMaskXor)
	if err != nil {
		return nil, err
	}
	state.SetPlayerStateMask(&rec, state.PlayerStateMask(rec)^b)

	b, err = rd.DecodeFreqByte(ft, freq.SVCPlayerInfoMotionMaskXor)
	if err != nil {
		return nil, err
	}
	state.SetPlayerMotionMask(&rec, state.PlayerMotionMask(rec)^b)

	packedOriginXY := rec[1]
	carriedOriginDeltaXY := rec[7]
	dx := int16(uint16(carriedOriginDeltaXY & 0xffff))
	dy := int16(uint16((carriedOriginDeltaXY >> 16) & 0xffff))
	packedOriginXY = packed.AddLow16(packedOriginXY, packed.Scaled16(dx, scale))
	packedOriginXY = packed.AddHigh16(packedOriginXY, packed.Scaled16(dy, scale))
	rec[1] = packedOriginXY

	packedOriginZ := rec[2]
	carriedOriginDeltaZ := rec[8]
	dz := int16(uint16(carriedOriginDeltaZ & 0xffff))
	packedOriginZ = packed.AddLow16(packedOriginZ, packed.Scaled16(dz, scale))
	rec[2] = packedOriginZ

	mask0 := state.PlayerOriginMask(rec)
	if (mask0 & 0x01) != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginXLoDelta)
		if err != nil {
			return nil, err
		}
		rec[1] = packed.AddLow16(rec[1], int16(int8(b)))
	}
	if (mask0 & 0x02) != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginXHiDelta)
		if err != nil {
			return nil, err
		}
		rec[1] = packed.AddLow16(rec[1], int16(uint16(b)<<8))
	}
	if (mask0 & 0x04) != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginYLoDelta)
		if err != nil {
			return nil, err
		}
		rec[1] = packed.AddHigh16(rec[1], int16(int8(b)))
	}
	if (mask0 & 0x08) != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginYHiDelta)
		if err != nil {
			return nil, err
		}
		rec[1] = packed.AddHigh16(rec[1], int16(uint16(b)<<8))
	}
	if (mask0 & 0x10) != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginZLoDelta)
		if err != nil {
			return nil, err
		}
		rec[2] = packed.AddLow16(rec[2], int16(int8(b)))
	}
	if (mask0 & 0x20) != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginZHiDelta)
		if err != nil {
			return nil, err
		}
		rec[2] = packed.AddLow16(rec[2], int16(uint16(b)<<8))
	}
	if (mask0 & 0x40) != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoOriginZByte2Delta)
		if err != nil {
			return nil, err
		}
		v := state.PlayerOriginZByte2(rec)
		v = byte(uint8(int(v) + int(int8(b))))
		state.SetPlayerOriginZByte2(&rec, v)
	}
	state.SetPlayerOriginMask(&rec, state.PlayerOriginMask(rec)&0xbf)

	stateMask := state.PlayerStateMask(rec)
	playerInfoFlags := byte(0)

	modelIndex := state.PlayerFreqByte(rec)
	if (stateMask & 0x20) != 0 {
		idx, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoFreqRemapIndex)
		if err != nil {
			return nil, err
		}
		modelIndex = st.ModelRemapByte(idx)
		state.SetPlayerFreqByte(&rec, modelIndex)
	}
	if modelIndex != st.PlayerFreqIndex {
		playerInfoFlags |= 0x20 // PF_MODEL
	}

	effectsByte := state.PlayerEffectsByte(rec)
	if (stateMask & 0x40) != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoEffectsSet)
		if err != nil {
			return nil, err
		}
		effectsByte = v
		state.SetPlayerEffectsByte(&rec, effectsByte)
	}
	if effectsByte != 0 {
		playerInfoFlags |= 0x40 // PF_EFFECTS
	}

	skinByte := state.PlayerSkinByte(rec)
	if int8(stateMask) < 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoSkinXor)
		if err != nil {
			return nil, err
		}
		skinByte ^= v
		state.SetPlayerSkinByte(&rec, skinByte)
	}
	if skinByte != 0 {
		playerInfoFlags |= 0x80 // PF_SKINNUM
	}

	state.SetPlayerStateMask(&rec, stateMask&0x97)

	motionMask := state.PlayerMotionMask(rec)
	accumulatorWordA := rec[10]
	if (motionMask & 0x01) != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelXLoDelta)
		if err != nil {
			return nil, err
		}
		accumulatorWordA = packed.AddLow16(accumulatorWordA, int16(int8(v)))
	}
	if (motionMask & 0x02) != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelXHiDelta)
		if err != nil {
			return nil, err
		}
		accumulatorWordA = packed.AddLow16(accumulatorWordA, int16(uint16(v)<<8))
	}
	rec[10] = accumulatorWordA

	carriedOriginDeltaXY = rec[7]
	prevLo18 := int16(uint16(carriedOriginDeltaXY & 0xffff))
	accLo := int16(uint16(accumulatorWordA & 0xffff))
	newLo18 := prevLo18 + accLo
	carriedOriginDeltaXY = packed.SetLow16(carriedOriginDeltaXY, newLo18)
	if newLo18 != 0 {
		playerInfoFlags |= 0x04 // PF_VELOCITY1
	}

	if (motionMask & 0x04) != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelYLoDelta)
		if err != nil {
			return nil, err
		}
		accumulatorWordA = packed.AddHigh16(accumulatorWordA, int16(int8(v)))
	}
	if (motionMask & 0x08) != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelYHiDelta)
		if err != nil {
			return nil, err
		}
		accumulatorWordA = packed.AddHigh16(accumulatorWordA, int16(uint16(v)<<8))
	}
	rec[10] = accumulatorWordA

	prevHi18 := int16(uint16((carriedOriginDeltaXY >> 16) & 0xffff))
	accHi := int16(uint16((accumulatorWordA >> 16) & 0xffff))
	newHi18 := prevHi18 + accHi
	carriedOriginDeltaXY = packed.SetHigh16(carriedOriginDeltaXY, newHi18)
	if newHi18 != 0 {
		playerInfoFlags |= 0x08 // PF_VELOCITY2
	}
	rec[7] = carriedOriginDeltaXY

	accumulatorWordB := rec[11]
	if (motionMask & 0x10) != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelZLoDelta)
		if err != nil {
			return nil, err
		}
		accumulatorWordB = packed.AddLow16(accumulatorWordB, int16(int8(v)))
	}
	if (motionMask & 0x20) != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoVelZHiDelta)
		if err != nil {
			return nil, err
		}
		accumulatorWordB = packed.AddLow16(accumulatorWordB, int16(uint16(v)<<8))
	}
	rec[11] = accumulatorWordB

	carriedOriginDeltaZ = rec[8]
	prevLo14 := int16(uint16(carriedOriginDeltaZ & 0xffff))
	accLo14 := int16(uint16(accumulatorWordB & 0xffff))
	newLo14 := prevLo14 + accLo14
	carriedOriginDeltaZ = packed.SetLow16(carriedOriginDeltaZ, newLo14)
	if newLo14 != 0 {
		playerInfoFlags |= 0x10 // PF_VELOCITY3
	}

	if (motionMask & 0x40) != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoWeaponFrameDelta)
		if err != nil {
			return nil, err
		}
		b2 := byte((carriedOriginDeltaZ >> 16) & 0xff)
		b2 = byte(uint8(int(b2) + int(int8(v))))
		carriedOriginDeltaZ = (carriedOriginDeltaZ &^ 0x00ff0000) | (uint32(b2) << 16)
	}

	state.SetPlayerRecordByte(&rec, 0x03, motionMask&0xbf)
	rec[8] = carriedOriginDeltaZ

	out = append(out, playerInfoFlags)

	b2 := state.PlayerRecordByte(rec, 0x22)
	extraPlayerInfoFlags := byte(0)
	if b2 != 0 {
		extraPlayerInfoFlags |= 0x01
	}
	if int8(motionMask) < 0 {
		extraPlayerInfoFlags |= 0x02
	}
	out = append(out, extraPlayerInfoFlags)

	packedOriginXY = rec[1]
	out = appendUint32LE(out, packedOriginXY)

	packedOriginZ = rec[2]
	out = append(out,
		byte(packedOriginZ&0xff),
		byte((packedOriginZ>>8)&0xff),
		byte((packedOriginZ>>16)&0xff),
	)

	if (playerInfoFlags & 0x04) != 0 {
		lo := uint16(rec[7] & 0xffff)
		out = appendUint16LE(out, lo)
	}
	if (playerInfoFlags & 0x08) != 0 {
		hi := uint16((rec[7] >> 16) & 0xffff)
		out = appendUint16LE(out, hi)
	}
	if (playerInfoFlags & 0x10) != 0 {
		lo := uint16(rec[8] & 0xffff)
		out = appendUint16LE(out, lo)
	}
	if (playerInfoFlags & 0x20) != 0 {
		out = append(out, state.PlayerRecordByte(rec, 0x19))
	}
	if (playerInfoFlags & 0x40) != 0 {
		out = append(out, state.PlayerRecordByte(rec, 0x1a))
	}
	if (playerInfoFlags & 0x80) != 0 {
		out = append(out, state.PlayerRecordByte(rec, 0x1b))
	}

	if (extraPlayerInfoFlags & 0x01) != 0 {
		out = append(out, b2)
	}
	st.CurrentPlayers = append(st.CurrentPlayers, rec)

	return out, nil
}

func (d *decoder) decodeSVCPlayerInfoDeltas(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	primaryPlayerPosXY := d.primaryPlayerPosXY
	primaryPlayerPosZ := d.primaryPlayerPosZ
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
		if player > 0x1f {
			return nil, fmt.Errorf("invalid svc_playerinfo player %d", player)
		}

		baseRec := basePlayerInfoRecord(
			basePlayers,
			player,
			primaryPlayerPosXY,
			primaryPlayerPosZ,
			st.PlayerFreqIndex,
		)

		recB := state.PlayerRecordBytesLE(baseRec)

		b, err := rd.DecodeFreqByte(ft, freq.SVCPlayerInfoMsec)
		if err != nil {
			return nil, err
		}
		recB[46] = b
		if int8(b) < 0 {
			recB[46] &= 0x7f
		} else {
			scaleStep := int(baseRec[6] & 0xff)
			scale := scaleStep
			if scaleStep != 0 {
				target := int(b) + (packetScale - int(state.PlayerRecordByte(baseRec, 0x2e)))
				for scale < target-scaleStep {
					scale += scaleStep
				}
			}
			dx := int16(state.GetU16LE(&recB, 28))
			dy := int16(state.GetU16LE(&recB, 30))
			dz := int16(state.GetU16LE(&recB, 32))
			addPlayerRecordI16(&recB, 4, packed.Scaled16(dx, scale))
			addPlayerRecordI16(&recB, 6, packed.Scaled16(dy, scale))
			addPlayerRecordI16(&recB, 8, packed.Scaled16(dz, scale))
			if err := decodePlayerInfoMaskDeltas(rd, ft, &recB); err != nil {
				return nil, err
			}
			if err := decodePlayerInfoOriginDeltas(rd, ft, &recB); err != nil {
				return nil, err
			}
			if err := decodePlayerInfoAngleDeltas(rd, ft, &recB); err != nil {
				return nil, err
			}
			if err := decodePlayerInfoStateDeltas(rd, ft, st, &recB); err != nil {
				return nil, err
			}
			if err := decodePlayerInfoVelocityDeltas(rd, ft, &recB); err != nil {
				return nil, err
			}
		}

		flags, extraFlags := buildPlayerInfoFlags(&recB, st.PlayerFreqIndex)

		if !firstPlayerInfo {
			out = append(out, 0x2a)
		}
		firstPlayerInfo = false
		out = appendPlayerInfoRecord(out, player, flags, extraFlags, &recB)

		st.CurrentPlayers = append(
			st.CurrentPlayers,
			state.PlayerRecordFromBytesLE(recB),
		)
	}
}

func (d *decoder) decodeSVCUpdatePing(out []byte) ([]byte, error) {
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

func (d *decoder) decodeSVCUpdatePL(out []byte) ([]byte, error) {
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
