package message

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/standard"
	"github.com/osm/quake/qizmo/state"
)

func (e *Encoder) parseFullPacketEntities(
	data []byte,
) ([]state.EntityRecord, int, bool) {
	if len(data) == 0 || data[0] != protocol.SVCPacketEntities {
		return nil, 0, false
	}
	off := 1
	var entities []state.EntityRecord
	lastEntity := uint16(maxPlayers)
	for {
		if off+2 > len(data) {
			return nil, 0, false
		}
		header := binary.LittleEndian.Uint16(data[off : off+2])
		off += 2
		if header == 0 {
			break
		}
		entityNumber := header & protocol.UCheckMoreBits
		if entityNumber <= lastEntity {
			return nil, 0, false
		}
		lastEntity = entityNumber
		bits := header &^ protocol.UCheckMoreBits
		if bits&protocol.UMoreBits != 0 {
			if off >= len(data) {
				return nil, 0, false
			}
			bits |= uint16(data[off])
			off++
		}
		if bits&protocol.URemove != 0 {
			return nil, 0, false
		}

		record := entityBaseline(e.state, entityNumber)
		state.SetEntityNumber(&record, entityNumber)
		clear(record[wire.EntityOriginCarryOffset:wire.EntityCarryEnd])
		for _, field := range wire.PacketEntityFields {
			if bits&field.Mask == 0 {
				continue
			}
			if off+field.Size > len(data) {
				return nil, 0, false
			}
			if field.Size == 1 {
				state.SetEntityRecordByte(&record, field.RecordOffset, data[off])
			} else {
				state.SetEntityRecordUint16(&record, field.RecordOffset, binary.LittleEndian.Uint16(data[off:off+2]))
			}
			off += field.Size
		}
		entities = append(entities, record)
	}

	canonical := append([]byte{protocol.SVCPacketEntities}, serializeSVCPacketEntitiesFull(e.state, entities)...)
	if !e.qwdInput && !bytes.Equal(data[:off], canonical) {
		return nil, 0, false
	}
	return entities, off, true
}

func (e *Encoder) encodeSVCPacketEntities(
	enc *rangeenc.Encoder,
	actions []packetEntityAction,
) error {
	currentEntity := uint16(maxPlayers)
	for i, action := range actions {
		switch action.kind {
		case packetEntityNew:
			if err := e.encodeNewPacketEntity(enc, action.target, &currentEntity); err != nil {
				return fmt.Errorf("new entity action %d: %w", i, err)
			}
		case packetEntityRemove:
			if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntitySymbol, byte(action.run)); err != nil {
				return fmt.Errorf("remove entity action %d: %w", i, err)
			}
			currentEntity = state.EntityNumber(action.base)
		case packetEntityDelta:
			if err := e.encodeDeltaPacketEntity(enc, action); err != nil {
				return fmt.Errorf("delta entity action %d: %w", i, err)
			}
			currentEntity = state.EntityNumber(action.base)
		}
	}
	return enc.EncodeFreqByte(e.ft, freq.SVCPacketEntitySymbol, 0)
}

type packetEntityActionKind byte

const (
	packetEntityNew packetEntityActionKind = iota
	packetEntityRemove
	packetEntityDelta
)

type packetEntityAction struct {
	kind   packetEntityActionKind
	run    int
	base   state.EntityRecord
	target state.EntityRecord
	plan   entityDeltaPlan
}

type packetEntityPlan struct {
	actions []packetEntityAction
	records []state.EntityRecord
}

func planPacketEntities(
	baseEntities []state.EntityRecord,
	entities []state.EntityRecord,
) packetEntityPlan {
	var actions []packetEntityAction
	result := make([]state.EntityRecord, 0, len(entities))
	baseIndex, targetIndex := 0, 0

	for {
		equal := 0
		for baseIndex+equal < len(baseEntities) && targetIndex+equal < len(entities) &&
			entityWireEqual(baseEntities[baseIndex+equal], entities[targetIndex+equal]) {
			equal++
		}
		if baseIndex+equal == len(baseEntities) && targetIndex+equal == len(entities) {
			result = append(result, baseEntities[baseIndex:]...)
			return packetEntityPlan{actions: actions, records: result}
		}

		baseAt := baseIndex + equal
		targetAt := targetIndex + equal
		if targetAt < len(entities) &&
			(baseAt == len(baseEntities) || state.EntityNumber(entities[targetAt]) < state.EntityNumber(baseEntities[baseAt])) {
			// A new-entity symbol implicitly copies every preceding base entity,
			// so Qizmo does not need to split a long unchanged run first.
			result = append(result, baseEntities[baseIndex:baseAt]...)
			record := entities[targetAt]
			state.SetEntityMask(&record, 0)
			clear(record[wire.EntityOriginCarryOffset:wire.EntityCarryEnd])
			actions = append(actions, packetEntityAction{kind: packetEntityNew, target: entities[targetAt]})
			result = append(result, record)
			baseIndex = baseAt
			targetIndex = targetAt + 1
			continue
		}
		if baseAt < len(baseEntities) &&
			(targetAt == len(entities) || state.EntityNumber(baseEntities[baseAt]) < state.EntityNumber(entities[targetAt])) {
			result = append(result, baseEntities[baseIndex:baseAt]...)
			actions = append(actions, packetEntityAction{
				kind: packetEntityRemove, run: equal + 1, base: baseEntities[baseAt],
			})
			baseIndex = baseAt + 1
			targetIndex = targetAt
			continue
		}

		result = append(result, baseEntities[baseIndex:baseAt]...)
		plan := planDeltaPacketEntity(baseEntities[baseAt], entities[targetAt])
		actions = append(actions, packetEntityAction{
			kind: packetEntityDelta, run: equal + 1,
			base: baseEntities[baseAt], target: entities[targetAt], plan: plan,
		})
		result = append(result, plan.result)
		baseIndex = baseAt + 1
		targetIndex = targetAt + 1
	}
}

type entityDeltaPlan struct {
	lowMask       byte
	highMask      byte
	effectiveHigh byte
	highXOR       byte
	lowXOR        byte
	originDelta   [3]packed.WordDelta
	carryDelta    [3]byte
	result        state.EntityRecord
}

func planDeltaPacketEntity(base, target state.EntityRecord) entityDeltaPlan {
	var plan entityDeltaPlan
	for _, field := range packetEntityFields {
		if field.offset >= wire.EntityAngleOffset {
			continue
		}
		if target[field.offset] != base[field.offset] {
			plan.lowMask |= field.mask
		}
	}
	result := base
	copy(
		result[wire.EntityVisibleStateOffset:wire.EntityVisibleStateEnd],
		target[wire.EntityVisibleStateOffset:wire.EntityVisibleStateEnd],
	)
	for _, field := range packetEntityFields {
		if field.offset < wire.EntityAngleOffset {
			continue
		}
		carryIndex := field.offset - wire.EntityAngleOffset
		carryOffset := wire.EntityAngleCarryOffset + carryIndex
		newCarry := target[field.offset] - base[field.offset]
		if newCarry != base[carryOffset] {
			plan.lowMask |= field.mask
			plan.carryDelta[carryIndex] = newCarry - base[carryOffset]
		}
		result[carryOffset] = newCarry
	}
	baseOrigin := state.EntityOrigin(base)
	targetOrigin := state.EntityOrigin(target)
	for axis := range plan.originDelta {
		newCarry := int16(targetOrigin[axis] - baseOrigin[axis])
		carryOff := wire.EntityOriginCarryOffset + axis*2
		baseCarry := int16(binary.LittleEndian.Uint16(base[carryOff : carryOff+2]))
		plan.originDelta[axis] = packed.SplitWordDelta(newCarry, baseCarry)
		setWordDeltaMask(&plan.highMask, plan.originDelta[axis], axis)
		binary.LittleEndian.PutUint16(result[carryOff:carryOff+2], uint16(newCarry))
	}
	baseMask := state.EntityMask(base)
	baseLow := byte(baseMask)
	baseHigh := byte(baseMask >> 8)
	plan.effectiveHigh = plan.highMask
	if plan.lowMask != baseLow {
		plan.effectiveHigh |= packetEntityLowMaskFlag
		plan.lowXOR = plan.lowMask ^ baseLow
	}
	plan.highXOR = plan.effectiveHigh ^ baseHigh
	state.SetEntityMask(&result, uint16(plan.lowMask)|uint16(plan.highMask)<<8)
	plan.result = result
	return plan
}

func (e *Encoder) encodeDeltaPacketEntity(enc *rangeenc.Encoder, action packetEntityAction) error {
	plan := action.plan
	if action.run > packetEntityMaxDeltaRun {
		return fmt.Errorf("delta run %d exceeds Qizmo maximum", action.run)
	}
	symbol := byte(packetEntityPayloadFlag)
	if action.run < packetEntityRunExtensionBase {
		symbol |= byte(action.run)
	}
	if plan.highXOR != 0 {
		symbol |= packetEntityHighMaskFlag
	}
	if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntitySymbol, symbol); err != nil {
		return err
	}
	if action.run >= packetEntityRunExtensionBase {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntitySymbol, byte(action.run-packetEntityRunExtensionBase)); err != nil {
			return err
		}
	}
	if plan.highXOR != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntityMaskHiXOR, plan.highXOR); err != nil {
			return err
		}
	}
	if plan.effectiveHigh&packetEntityLowMaskFlag != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntityMaskLoXOR, plan.lowXOR); err != nil {
			return err
		}
	}
	base, target := action.base, action.target
	for _, field := range packetEntityFields {
		if plan.lowMask&field.mask == 0 {
			continue
		}
		var value byte
		switch {
		case field.kind == packetEntityFieldModel:
			remapIndex, ok := e.state.ModelRemapIndex(target[field.offset])
			if !ok {
				return fmt.Errorf("model %d has no remap", target[field.offset])
			}
			value = remapIndex
		case field.kind == packetEntityFieldXOR:
			value = target[field.offset] ^ base[field.offset]
		case field.offset >= wire.EntityAngleOffset:
			value = plan.carryDelta[field.offset-wire.EntityAngleOffset]
		default:
			value = target[field.offset] - base[field.offset]
		}
		if err := enc.EncodeFreqByte(e.ft, field.row, value); err != nil {
			return err
		}
	}
	for i, rows := range packetEntityOriginDeltaRows {
		if err := e.encodeWordDelta(enc, plan.originDelta[i], rows.low, rows.high); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeNewPacketEntity(
	enc *rangeenc.Encoder,
	record state.EntityRecord,
	currentEntity *uint16,
) error {
	entityNumber := state.EntityNumber(record)
	if entityNumber <= *currentEntity {
		return fmt.Errorf("entity %d is not after %d", entityNumber, *currentEntity)
	}
	base := entityBaseline(e.state, entityNumber)
	var lowMask byte
	for _, field := range packetEntityFields {
		if record[field.offset] != base[field.offset] {
			lowMask |= field.mask
		}
	}
	recordOrigin := state.EntityOrigin(record)
	baseOrigin := state.EntityOrigin(base)
	var originDelta [3]packed.WordDelta
	for axis := range originDelta {
		originDelta[axis] = packed.SplitWordDelta(
			int16(recordOrigin[axis]),
			int16(baseOrigin[axis]),
		)
	}
	var highMask byte
	for axis, delta := range originDelta {
		setWordDeltaMask(&highMask, delta, axis)
	}
	if lowMask != 0 {
		highMask |= packetEntityLowMaskFlag
	}
	difference := entityNumber - *currentEntity
	symbol := byte(packetEntityNewFlag)
	if highMask != 0 {
		symbol |= packetEntityPayloadFlag
	}
	if difference < packetEntityNumberExtensionBase {
		symbol |= byte(difference)
	}
	if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntitySymbol, symbol); err != nil {
		return err
	}
	if difference >= packetEntityNumberExtensionBase {
		remaining := difference - packetEntityNumberExtensionBase
		for remaining >= packetEntityExtensionChunk {
			if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntityNumDeltaExt, packetEntityExtensionChunk); err != nil {
				return err
			}
			remaining -= packetEntityExtensionChunk
		}
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntityNumDeltaExt, byte(remaining)); err != nil {
			return err
		}
	}
	*currentEntity = entityNumber
	if highMask == 0 {
		return nil
	}
	if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntityMaskHiXOR, highMask); err != nil {
		return err
	}
	if lowMask != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPacketEntityMaskLoXOR, lowMask); err != nil {
			return err
		}
	}
	for _, field := range packetEntityFields {
		if lowMask&field.mask == 0 {
			continue
		}
		value := record[field.offset] - base[field.offset]
		if field.kind == packetEntityFieldModel {
			remapIndex, ok := e.state.ModelRemapIndex(record[field.offset])
			if !ok {
				return fmt.Errorf("model %d has no remap", record[field.offset])
			}
			value = remapIndex
		} else if field.kind == packetEntityFieldXOR {
			value = record[field.offset] ^ base[field.offset]
		}
		if err := enc.EncodeFreqByte(e.ft, field.row, value); err != nil {
			return err
		}
	}
	for i, rows := range packetEntityOriginDeltaRows {
		if err := e.encodeWordDelta(enc, originDelta[i], rows.low, rows.high); err != nil {
			return err
		}
	}
	return nil
}

// parseDeltaPacketEntities expands an ordinary QuakeWorld entity delta into
// the full snapshot consumed by the message encoder. The original delta is
// accepted only in the canonical form produced by the matching decoder.
func (e *Encoder) parseDeltaPacketEntities(
	data []byte,
) ([]state.EntityRecord, byte, int, bool) {
	if len(data) < 4 || data[0] != protocol.SVCDeltaPacketEntities {
		return nil, 0, 0, false
	}
	baseSequence := data[1]
	base, ok := e.state.RawEntitySnapshot(baseSequence)
	if !ok {
		return nil, 0, 0, false
	}

	entities, consumed, err := standard.ParsePacketEntities(e.state, data[2:], base)
	if err != nil {
		return nil, 0, 0, false
	}
	offset := 2 + consumed

	baseRecords := orderedEntityRecords(base)
	entityRecords := orderedEntityRecords(entities)
	canonical := []byte{protocol.SVCDeltaPacketEntities, baseSequence}
	canonical = append(canonical, serializeSVCDeltaPacketEntities(e.state, baseRecords, entityRecords)...)
	if !bytes.Equal(data[:offset], canonical) {
		return nil, 0, 0, false
	}
	return entityRecords, baseSequence, offset, true
}

func orderedEntityRecords(records map[uint16]state.EntityRecord) []state.EntityRecord {
	ordered := make([]state.EntityRecord, 0, len(records))
	for _, record := range records {
		ordered = append(ordered, record)
	}
	sort.Slice(ordered, func(i, j int) bool {
		return state.EntityNumber(ordered[i]) < state.EntityNumber(ordered[j])
	})
	return ordered
}
