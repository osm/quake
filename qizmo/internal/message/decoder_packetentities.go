package message

import (
	"fmt"

	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/state"
)

const entityOriginCarryOffset = 18

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
	for _, field := range packetEntityFields {
		if mask&uint16(field.mask) == 0 {
			continue
		}
		value, err := rd.DecodeFreqByte(ft, field.row)
		if err != nil {
			return err
		}
		switch field.kind {
		case packetEntityFieldModel:
			state.SetEntityRecordByte(rec, field.offset, st.ModelForRemapIndex(value))
		case packetEntityFieldXOR:
			state.SetEntityRecordByte(
				rec,
				field.offset,
				state.EntityRecordByte(*rec, field.offset)^value,
			)
		default:
			addEntityByteDelta(rec, field.offset, value)
		}
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
	offsets := [3]int{xOff, yOff, zOff}
	for axis, rows := range packetEntityPositionDeltaRows {
		lowMask := uint16(1 << uint(8+axis*2))
		delta, err := decodeMaskedWordDelta(rd, ft, mask, lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		state.AddEntityRecordInt16(rec, offsets[axis], delta)
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
	for _, field := range packetEntityFields {
		present := mask&uint16(field.mask) != 0
		if field.offset >= 9 {
			carryOffset := field.offset + 15
			if present {
				value, err := rd.DecodeFreqByte(ft, field.row)
				if err != nil {
					return err
				}
				addEntityByteDelta(rec, carryOffset, value)
			}
			addEntityByteDelta(rec, field.offset, state.EntityRecordByte(*rec, carryOffset))
			continue
		}
		if !present {
			continue
		}
		value, err := rd.DecodeFreqByte(ft, field.row)
		if err != nil {
			return err
		}
		switch field.kind {
		case packetEntityFieldModel:
			state.SetEntityRecordByte(rec, field.offset, st.ModelForRemapIndex(value))
		case packetEntityFieldXOR:
			state.SetEntityRecordByte(
				rec,
				field.offset,
				state.EntityRecordByte(*rec, field.offset)^value,
			)
		default:
			addEntityByteDelta(rec, field.offset, value)
		}
	}

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

type packetEntityStream struct {
	rd            *rangedec.Decoder
	ft            *freq.Tables
	state         *state.Packet
	base          []state.EntityRecord
	baseSequence  uint32
	baseIndex     int
	currentNumber uint16
	entities      []state.EntityRecord
}

func (d *packetDecoder) decodeSVCPacketEntities(preserveDelta bool) ([]byte, error) {
	var baseEntities []state.EntityRecord
	baseSequence, hasPacketBase := d.state.PacketBase()
	if hasPacketBase {
		baseEntities, _ = d.state.EntitySnapshot(baseSequence)
	}
	stream := packetEntityStream{
		rd:            d.rd,
		ft:            d.ft,
		state:         d.state,
		base:          baseEntities,
		baseSequence:  baseSequence,
		currentNumber: 0x20,
	}

	for {
		symbol, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPacketEntitySymbol)
		if err != nil {
			return nil, err
		}
		if symbol == 0 {
			break
		}
		if symbol&0x80 != 0 {
			err = stream.decodeNewEntity(symbol)
		} else {
			err = stream.decodeBaseEntity(symbol)
		}
		if err != nil {
			return nil, err
		}
	}

	stream.appendRemainingBase()
	return d.commitPacketEntities(baseEntities, stream.entities, preserveDelta)
}

func (s *packetEntityStream) decodeNewEntity(symbol byte) error {
	delta := uint16(symbol & 0x3f)
	if delta == 0 {
		s.currentNumber += 0x40
		for {
			value, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityNumDeltaExt)
			if err != nil {
				return err
			}
			if value != 0xff {
				delta = uint16(value)
				break
			}
			s.currentNumber += 0xff
		}
	}
	s.currentNumber += delta

	for s.baseIndex < len(s.base) && state.EntityNumber(s.base[s.baseIndex]) < s.currentNumber {
		s.entities = append(s.entities, s.base[s.baseIndex])
		s.baseIndex++
	}
	if s.baseIndex < len(s.base) && state.EntityNumber(s.base[s.baseIndex]) == s.currentNumber {
		return fmt.Errorf("duplicate packet entity %d", s.currentNumber)
	}

	record := entityBaseline(s.state, s.currentNumber)
	state.SetEntityNumber(&record, s.currentNumber)
	// Reset coordinate-delta carry for a new entity.
	for offset := 18; offset <= 26; offset++ {
		record[offset] = 0
	}

	if symbol&0x40 != 0 {
		highMask, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityMaskHiXOR)
		if err != nil {
			return err
		}
		mask := uint16(highMask) << 8
		if mask&0x4000 != 0 {
			lowMask, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityMaskLoXOR)
			if err != nil {
				return err
			}
			mask |= uint16(lowMask)
			if err := decodeNewEntityFieldDeltas(s.rd, s.ft, s.state, &record, uint16(lowMask)); err != nil {
				return err
			}
		}
		if err := decodeEntityPositionDeltas(s.rd, s.ft, &record, mask, 12, 14, 16); err != nil {
			return err
		}
	}

	record[2] = 0
	record[3] = 0
	s.entities = append(s.entities, record)
	return nil
}

func (s *packetEntityStream) decodeBaseEntity(symbol byte) error {
	run, err := decodeRunLength(s.rd, s.ft, symbol)
	if err != nil {
		return err
	}
	for run > 1 {
		if s.baseIndex >= len(s.base) {
			return fmt.Errorf(
				"%w: seq=%d svc_deltapacketentities copy overflow run=%d base=%d/%d baseSeq=%d",
				errDroppedPacket,
				s.state.Sequence(),
				run,
				s.baseIndex,
				len(s.base),
				s.baseSequence,
			)
		}
		s.entities = append(s.entities, s.base[s.baseIndex])
		s.baseIndex++
		run--
	}
	if s.baseIndex >= len(s.base) {
		return fmt.Errorf(
			"%w: seq=%d svc_deltapacketentities missing base sym=0x%02x base=%d/%d baseSeq=%d",
			errDroppedPacket,
			s.state.Sequence(),
			symbol,
			s.baseIndex,
			len(s.base),
			s.baseSequence,
		)
	}

	s.currentNumber = state.EntityNumber(s.base[s.baseIndex])
	if symbol&0x40 == 0 {
		s.baseIndex++
		return nil
	}

	record := s.base[s.baseIndex]
	s.baseIndex++
	mask := uint16(record[2]) | uint16(record[3])<<8
	if symbol&0x20 != 0 {
		value, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityMaskHiXOR)
		if err != nil {
			return err
		}
		mask ^= uint16(value) << 8
	}
	if mask&0x4000 != 0 {
		value, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityMaskLoXOR)
		if err != nil {
			return err
		}
		mask ^= uint16(value)
	}
	if err := decodeDeltaEntityFieldDeltas(s.rd, s.ft, s.state, &record, mask); err != nil {
		return err
	}
	if err := decodeEntityPositionDeltas(s.rd, s.ft, &record, mask, 18, 20, 22); err != nil {
		return err
	}
	origin := state.EntityOrigin(record)
	for axis := range origin {
		origin[axis] += state.EntityRecordUint16(record, entityOriginCarryOffset+axis*2)
	}
	state.SetEntityOrigin(&record, origin)
	// Retain the effective mask for later delta steps.
	record[2] = byte(mask)
	record[3] = byte(mask>>8) & 0x3f
	s.entities = append(s.entities, record)
	return nil
}

func (s *packetEntityStream) appendRemainingBase() {
	for s.baseIndex < len(s.base) {
		s.entities = append(s.entities, s.base[s.baseIndex])
		s.baseIndex++
	}
}

func (d *packetDecoder) commitPacketEntities(
	baseEntities []state.EntityRecord,
	packetEntities []state.EntityRecord,
	preserveDelta bool,
) ([]byte, error) {
	st := d.state
	st.CommitPacketEntities(packetEntities, preserveDelta)
	baseSequence, hasPacketBase := st.PacketBase()
	if preserveDelta && hasPacketBase {
		body := []byte{byte(baseSequence)}
		return append(body, serializeSVCDeltaPacketEntities(st, baseEntities, packetEntities)...), nil
	}
	return serializeSVCPacketEntitiesFull(st, packetEntities), nil
}
