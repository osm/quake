package compressed

import (
	"fmt"

	"github.com/osm/quake/demo/qwz/freq"
	"github.com/osm/quake/demo/qwz/rangedec"
	"github.com/osm/quake/demo/qwz/state"
)

func addEntityByteDelta(rec *state.EntityRecord, field int, value byte) {
	next := int8(state.EntityRecordByte(*rec, field)) + int8(value)
	state.SetEntityRecordByte(rec, field, byte(next))
}

func decodeNewEntityFieldDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	st *state.Packet,
	rec *state.EntityRecord,
	mask uint16,
) error {
	if mask&0x01 != 0 {
		idx, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityFreqIndex)
		if err != nil {
			return err
		}
		state.SetEntityRecordByte(rec, 4, st.ModelRemapByte(idx))
	}
	if mask&0x02 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField5Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 5, value)
	}
	if mask&0x04 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField6Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 6, value)
	}
	if mask&0x08 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField7Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 7, value)
	}
	if mask&0x10 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField8Xor)
		if err != nil {
			return err
		}
		state.SetEntityRecordByte(
			rec,
			8,
			state.EntityRecordByte(*rec, 8)^value,
		)
	}
	if mask&0x20 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField9Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 9, value)
	}
	if mask&0x40 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField10Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 10, value)
	}
	if mask&0x80 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField11Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 11, value)
	}

	return nil
}

func decodeEntityPositionDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	rec *state.EntityRecord,
	mask uint16,
	xOff, yOff, zOff int,
) error {
	if mask&0x0100 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityPosXLoDelta)
		if err != nil {
			return err
		}
		state.AddEntityRecordI16(rec, xOff, int16(int8(value)))
	}
	if mask&0x0200 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityPosXHiDelta)
		if err != nil {
			return err
		}
		state.AddEntityRecordI16(rec, xOff, int16(uint16(value)<<8))
	}
	if mask&0x0400 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityPosYLoDelta)
		if err != nil {
			return err
		}
		state.AddEntityRecordI16(rec, yOff, int16(int8(value)))
	}
	if mask&0x0800 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityPosYHiDelta)
		if err != nil {
			return err
		}
		state.AddEntityRecordI16(rec, yOff, int16(uint16(value)<<8))
	}
	if mask&0x1000 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityPosZLoDelta)
		if err != nil {
			return err
		}
		state.AddEntityRecordI16(rec, zOff, int16(int8(value)))
	}
	if mask&0x2000 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityPosZHiDelta)
		if err != nil {
			return err
		}
		state.AddEntityRecordI16(rec, zOff, int16(uint16(value)<<8))
	}

	return nil
}

func decodeDeltaEntityFieldDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	st *state.Packet,
	rec *state.EntityRecord,
	mask uint16,
) error {
	// Bytes 24..26 carry the hidden running deltas that feed wire fields 9..11.
	if mask&0x0001 != 0 {
		idx, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityFreqIndex)
		if err != nil {
			return err
		}
		state.SetEntityRecordByte(rec, 4, st.ModelRemapByte(idx))
	}
	if mask&0x0002 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField5Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 5, value)
	}
	if mask&0x0004 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField6Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 6, value)
	}
	if mask&0x0008 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField7Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 7, value)
	}
	if mask&0x0010 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField8Xor)
		if err != nil {
			return err
		}
		state.SetEntityRecordByte(
			rec,
			8,
			state.EntityRecordByte(*rec, 8)^value,
		)
	}
	if mask&0x0020 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField9Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 24, value)
	}
	addEntityByteDelta(rec, 9, state.EntityRecordByte(*rec, 24))

	if mask&0x0040 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField10Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 25, value)
	}
	addEntityByteDelta(rec, 10, state.EntityRecordByte(*rec, 25))

	if mask&0x0080 != 0 {
		value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityField11Delta)
		if err != nil {
			return err
		}
		addEntityByteDelta(rec, 26, value)
	}
	addEntityByteDelta(rec, 11, state.EntityRecordByte(*rec, 26))

	return nil
}

func decodeRunLength(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	sym byte,
) (int, error) {
	run := int(sym)
	if sym&0x40 == 0 {
		return run, nil
	}

	run = int(sym & 0x1f)
	if run != 0 {
		return run, nil
	}

	value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntitySymbol)
	if err != nil {
		return 0, err
	}

	return int(value) + 0x20, nil
}

func serializeSVCPacketEntitiesFull(
	st *state.Packet,
	ents []state.EntityRecord,
) []byte {
	out := make([]byte, 0, len(ents)*8+2)
	for _, rec := range ents {
		entNum := state.EntityNumber(rec)
		base := entityBaseline(st, entNum)
		bits := uint16(0)
		if state.EntityRecordByte(rec, 4) != state.EntityRecordByte(base, 4) {
			bits |= 0x0004
		}
		if state.EntityRecordByte(rec, 5) != state.EntityRecordByte(base, 5) {
			bits |= 0x2000
		}
		if state.EntityRecordByte(rec, 6) != state.EntityRecordByte(base, 6) {
			bits |= 0x0008
		}
		if state.EntityRecordByte(rec, 7) != state.EntityRecordByte(base, 7) {
			bits |= 0x0010
		}
		if state.EntityRecordByte(rec, 8) != state.EntityRecordByte(base, 8) {
			bits |= 0x0020
		}
		if state.EntityRecordByte(rec, 9) != state.EntityRecordByte(base, 9) {
			bits |= 0x0001
		}
		if state.EntityRecordByte(rec, 10) != state.EntityRecordByte(base, 10) {
			bits |= 0x1000
		}
		if state.EntityRecordByte(rec, 11) != state.EntityRecordByte(base, 11) {
			bits |= 0x0002
		}
		if state.EntityRecordU16(rec, 12) != state.EntityRecordU16(base, 12) {
			bits |= 0x0200
		}
		if state.EntityRecordU16(rec, 14) != state.EntityRecordU16(base, 14) {
			bits |= 0x0400
		}
		if state.EntityRecordU16(rec, 16) != state.EntityRecordU16(base, 16) {
			bits |= 0x0800
		}
		if bits&0x00ff != 0 {
			bits |= 0x8000
		}
		out = append(out, byte(entNum&0xff), byte((entNum>>8)&0x01)|byte(bits>>8))
		if bits&0x8000 != 0 {
			out = append(out, byte(bits))
		}
		if bits&0x0004 != 0 {
			out = append(out, state.EntityRecordByte(rec, 4))
		}
		if bits&0x2000 != 0 {
			out = append(out, state.EntityRecordByte(rec, 5))
		}
		if bits&0x0008 != 0 {
			out = append(out, state.EntityRecordByte(rec, 6))
		}
		if bits&0x0010 != 0 {
			out = append(out, state.EntityRecordByte(rec, 7))
		}
		if bits&0x0020 != 0 {
			out = append(out, state.EntityRecordByte(rec, 8))
		}
		if bits&0x0200 != 0 {
			v := state.EntityRecordU16(rec, 12)
			out = appendUint16LE(out, v)
		}
		if bits&0x0001 != 0 {
			out = append(out, state.EntityRecordByte(rec, 9))
		}
		if bits&0x0400 != 0 {
			v := state.EntityRecordU16(rec, 14)
			out = appendUint16LE(out, v)
		}
		if bits&0x1000 != 0 {
			out = append(out, state.EntityRecordByte(rec, 10))
		}
		if bits&0x0800 != 0 {
			v := state.EntityRecordU16(rec, 16)
			out = appendUint16LE(out, v)
		}
		if bits&0x0002 != 0 {
			out = append(out, state.EntityRecordByte(rec, 11))
		}
	}
	out = append(out, 0, 0)
	return out
}

func (d *decoder) decodeSVCPacketEntitiesFull() ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state

	var baseEntities []state.EntityRecord
	if st.PacketHasBase {
		baseEntities, _ = st.FindEntHistoryBySeq(st.PacketBaseSeq)
	}
	baseIndex := 0
	currentEntityNum := uint16(0x20)
	packetEntities := make([]state.EntityRecord, 0, 64)

	for {
		sym, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntitySymbol)
		if err != nil {
			return nil, err
		}
		if sym == 0 {
			break
		}

		if sym&0x80 != 0 {
			delta := uint16(sym & 0x3f)
			if delta == 0 {
				currentEntityNum += 0x40
				for {
					b, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityNumDeltaExt)
					if err != nil {
						return nil, err
					}
					if b != 0xff {
						delta = uint16(b)
						break
					}
					currentEntityNum += 0xff
				}
			}
			currentEntityNum += delta

			for baseIndex < len(baseEntities) &&
				state.EntityNumber(baseEntities[baseIndex]) < currentEntityNum {
				packetEntities = append(packetEntities, baseEntities[baseIndex])
				baseIndex++
			}
			if baseIndex < len(baseEntities) &&
				state.EntityNumber(baseEntities[baseIndex]) == currentEntityNum {
				return nil, fmt.Errorf("duplicate packet entity %d", currentEntityNum)
			}

			rec := entityBaseline(st, currentEntityNum)
			state.SetEntityNumber(&rec, currentEntityNum)
			for i := 18; i <= 26; i++ {
				rec[i] = 0
			}

			if sym&0x40 != 0 {
				mask := uint16(0)
				v, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityMaskHiXor)
				if err != nil {
					return nil, err
				}
				mask = uint16(v) << 8
				if mask&0x4000 != 0 {
					ext, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityMaskLoXor)
					if err != nil {
						return nil, err
					}
					mask |= uint16(ext)
					if err := decodeNewEntityFieldDeltas(
						rd,
						ft,
						st,
						&rec,
						uint16(ext),
					); err != nil {
						return nil, err
					}
				}
				if err := decodeEntityPositionDeltas(
					rd,
					ft,
					&rec,
					mask,
					12,
					14,
					16,
				); err != nil {
					return nil, err
				}
			}

			rec[2] = 0
			rec[3] = 0
			packetEntities = append(packetEntities, rec)
			continue
		}

		run, err := decodeRunLength(rd, ft, sym)
		if err != nil {
			return nil, err
		}
		for run > 1 {
			if baseIndex >= len(baseEntities) {
				return nil, fmt.Errorf(
					"%w: "+
						"seq=%d 0x30 copy overflow run=%d base=%d/%d seq=%d",
					errDroppedPacket,
					st.SeqNo(),
					run,
					baseIndex,
					len(baseEntities),
					st.PacketBaseSeq,
				)
			}
			packetEntities = append(packetEntities, baseEntities[baseIndex])
			baseIndex++
			run--
		}
		if baseIndex >= len(baseEntities) {
			return nil, fmt.Errorf(
				"%w: "+
					"seq=%d 0x30 missing base sym=0x%02x base=%d/%d seq=%d",
				errDroppedPacket,
				st.SeqNo(),
				sym,
				baseIndex,
				len(baseEntities),
				st.PacketBaseSeq,
			)
		}
		currentEntityNum = state.EntityNumber(baseEntities[baseIndex])
		if sym&0x40 == 0 {
			baseIndex++
			continue
		}

		mask := uint16(baseEntities[baseIndex][2]) | (uint16(baseEntities[baseIndex][3]) << 8)
		if sym&0x20 != 0 {
			v, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityMaskHiXor)
			if err != nil {
				return nil, err
			}
			mask ^= uint16(v) << 8
		}

		rec := baseEntities[baseIndex]
		baseIndex++

		if mask&0x4000 != 0 {
			ext, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntityMaskLoXor)
			if err != nil {
				return nil, err
			}
			mask ^= uint16(ext)
		}
		if err := decodeDeltaEntityFieldDeltas(rd, ft, st, &rec, mask); err != nil {
			return nil, err
		}
		if err := decodeEntityPositionDeltas(
			rd,
			ft,
			&rec,
			mask,
			18,
			20,
			22,
		); err != nil {
			return nil, err
		}
		state.AddEntityRecordI16(&rec, 12, int16(state.EntityRecordU16(rec, 18)))
		state.AddEntityRecordI16(&rec, 14, int16(state.EntityRecordU16(rec, 20)))
		state.AddEntityRecordI16(&rec, 16, int16(state.EntityRecordU16(rec, 22)))
		rec[2] = byte(mask)
		rec[3] = byte(mask>>8) & 0x3f
		packetEntities = append(packetEntities, rec)
	}

	for baseIndex < len(baseEntities) {
		packetEntities = append(packetEntities, baseEntities[baseIndex])
		baseIndex++
	}

	entitySet := make(map[uint16]state.EntityRecord, len(packetEntities))
	for _, rec := range packetEntities {
		entitySet[state.EntityNumber(rec)] = rec
	}
	st.CommitEntities(st.SeqNo(), entitySet)
	st.PacketEntsCommitted = true
	for entNum, rec := range entitySet {
		st.EntityLast[entNum] = rec
		st.EntityLastRaw[entNum] = false
	}
	body := serializeSVCPacketEntitiesFull(st, packetEntities)
	return body, nil
}

func entityBaseline(st *state.Packet, entNum uint16) state.EntityRecord {
	if rec, ok := st.Baselines[entNum]; ok {
		return rec
	}
	var rec state.EntityRecord
	state.SetEntityNumber(&rec, entNum)
	return rec
}
