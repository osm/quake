package message

import (
	"fmt"

	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/state"
)

func addEntityByteDelta(record *state.EntityRecord, field int, value byte) {
	next := int8(state.EntityRecordByte(*record, field)) + int8(value)
	state.SetEntityRecordByte(record, field, byte(next))
}

func decodeNewEntityFieldDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	st *state.Packet,
	record *state.EntityRecord,
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
			state.SetEntityRecordByte(record, field.offset, st.ModelForRemapIndex(value))
		case packetEntityFieldXOR:
			state.SetEntityRecordByte(
				record,
				field.offset,
				state.EntityRecordByte(*record, field.offset)^value,
			)
		default:
			addEntityByteDelta(record, field.offset, value)
		}
	}

	return nil
}

func decodeEntityOriginDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	record *state.EntityRecord,
	mask uint16,
	offset int,
) error {
	for axis, rows := range packetEntityOriginDeltaRows {
		lowMask := uint16(1 << uint(8+axis*2))
		delta, err := decodeMaskedWordDelta(rd, ft, mask, lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		state.AddEntityRecordInt16(record, offset+axis*2, delta)
	}

	return nil
}

func decodeDeltaEntityFieldDeltas(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	st *state.Packet,
	record *state.EntityRecord,
	mask uint16,
) error {
	for _, field := range packetEntityFields {
		present := mask&uint16(field.mask) != 0
		if field.offset >= wire.EntityAngleOffset {
			carryOffset := wire.EntityAngleCarryOffset + field.offset - wire.EntityAngleOffset
			if present {
				value, err := rd.DecodeFreqByte(ft, field.row)
				if err != nil {
					return err
				}
				addEntityByteDelta(record, carryOffset, value)
			}
			addEntityByteDelta(record, field.offset, state.EntityRecordByte(*record, carryOffset))
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
			state.SetEntityRecordByte(record, field.offset, st.ModelForRemapIndex(value))
		case packetEntityFieldXOR:
			state.SetEntityRecordByte(
				record,
				field.offset,
				state.EntityRecordByte(*record, field.offset)^value,
			)
		default:
			addEntityByteDelta(record, field.offset, value)
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
	if sym&packetEntityPayloadFlag == 0 {
		return run, nil
	}

	run = int(sym & packetEntityRunMask)
	if run != 0 {
		return run, nil
	}

	value, err := rd.DecodeFreqByte(ft, freq.SVCPacketEntitySymbol)
	if err != nil {
		return 0, err
	}

	return int(value) + packetEntityRunExtensionBase, nil
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
		currentNumber: maxPlayers,
	}

	for {
		symbol, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPacketEntitySymbol)
		if err != nil {
			return nil, err
		}
		if symbol == 0 {
			break
		}
		if symbol&packetEntityNewFlag != 0 {
			err = stream.decodeNewEntity(symbol)
		} else {
			err = stream.decodeBaseEntity(symbol)
		}
		if err != nil {
			return nil, err
		}
	}

	stream.appendRemainingBase()
	return d.finishPacketEntities(baseEntities, stream.entities, preserveDelta)
}

func (s *packetEntityStream) decodeNewEntity(symbol byte) error {
	delta := uint16(symbol & packetEntityNumberDeltaMask)
	if delta == 0 {
		s.currentNumber += packetEntityNumberExtensionBase
		for {
			value, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityNumDeltaExt)
			if err != nil {
				return err
			}
			if value != packetEntityExtensionChunk {
				delta = uint16(value)
				break
			}
			s.currentNumber += packetEntityExtensionChunk
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
	clear(record[wire.EntityOriginCarryOffset:wire.EntityCarryEnd])

	if symbol&packetEntityPayloadFlag != 0 {
		highMask, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityMaskHiXOR)
		if err != nil {
			return err
		}
		mask := uint16(highMask) << 8
		if packetEntityHasLowMask(mask) {
			lowMask, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityMaskLoXOR)
			if err != nil {
				return err
			}
			mask |= uint16(lowMask)
			if err := decodeNewEntityFieldDeltas(s.rd, s.ft, s.state, &record, uint16(lowMask)); err != nil {
				return err
			}
		}
		if err := decodeEntityOriginDeltas(s.rd, s.ft, &record, mask, wire.EntityOriginOffset); err != nil {
			return err
		}
	}

	state.SetEntityMask(&record, 0)
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
	if symbol&packetEntityPayloadFlag == 0 {
		s.baseIndex++
		return nil
	}

	record := s.base[s.baseIndex]
	s.baseIndex++
	mask := state.EntityMask(record)
	if symbol&packetEntityHighMaskFlag != 0 {
		value, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityMaskHiXOR)
		if err != nil {
			return err
		}
		mask ^= uint16(value) << 8
	}
	if packetEntityHasLowMask(mask) {
		value, err := s.rd.DecodeFreqByte(s.ft, freq.SVCPacketEntityMaskLoXOR)
		if err != nil {
			return err
		}
		mask ^= uint16(value)
	}
	if err := decodeDeltaEntityFieldDeltas(s.rd, s.ft, s.state, &record, mask); err != nil {
		return err
	}
	if err := decodeEntityOriginDeltas(s.rd, s.ft, &record, mask, wire.EntityOriginCarryOffset); err != nil {
		return err
	}
	origin := state.EntityOrigin(record)
	for axis := range 3 {
		carryOffset := wire.EntityOriginCarryOffset + axis*2
		origin[axis] += state.EntityRecordUint16(record, carryOffset)
	}
	state.SetEntityOrigin(&record, origin)
	// Retain the effective mask for later delta steps.
	state.SetEntityMask(&record, mask)
	s.entities = append(s.entities, record)
	return nil
}

func (s *packetEntityStream) appendRemainingBase() {
	for s.baseIndex < len(s.base) {
		s.entities = append(s.entities, s.base[s.baseIndex])
		s.baseIndex++
	}
}

func (d *packetDecoder) finishPacketEntities(
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
