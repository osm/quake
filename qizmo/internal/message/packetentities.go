package message

import (
	"bytes"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/state"
)

func entityWireEqual(a, b state.EntityRecord) bool {
	visibleA := a[wire.EntityVisibleStateOffset:wire.EntityVisibleStateEnd]
	visibleB := b[wire.EntityVisibleStateOffset:wire.EntityVisibleStateEnd]
	return state.EntityNumber(a) == state.EntityNumber(b) && bytes.Equal(visibleA, visibleB)
}

func serializeSVCPacketEntitiesFull(
	st *state.Packet,
	entities []state.EntityRecord,
) []byte {
	out := make([]byte, 0, len(entities)*8+2)
	for _, record := range entities {
		entityNumber := state.EntityNumber(record)
		base := entityBaseline(st, entityNumber)
		out = appendPacketEntityUpdate(out, entityNumber, packetEntityWireBits(base, record), record)
	}
	out = append(out, 0, 0)
	return out
}

func packetEntityWireBits(base, target state.EntityRecord) uint16 {
	bits := uint16(0)
	for _, field := range wire.PacketEntityFields {
		changed := state.EntityRecordByte(target, field.RecordOffset) !=
			state.EntityRecordByte(base, field.RecordOffset)
		if field.Size == 2 {
			changed = state.EntityRecordUint16(target, field.RecordOffset) !=
				state.EntityRecordUint16(base, field.RecordOffset)
		}
		if changed {
			bits |= field.Mask
		}
	}
	if bits&protocol.UCheckMoreBits != 0 {
		bits |= protocol.UMoreBits
	}
	return bits
}

func appendPacketEntityUpdate(
	out []byte,
	entityNumber uint16,
	bits uint16,
	target state.EntityRecord,
) []byte {
	header := entityNumber&protocol.UCheckMoreBits | bits&^protocol.UCheckMoreBits
	out = appendUint16LE(out, header)
	if bits&protocol.UMoreBits != 0 {
		out = append(out, byte(bits))
	}
	for _, field := range wire.PacketEntityFields {
		if bits&field.Mask == 0 {
			continue
		}
		if field.Size == 1 {
			out = append(out, state.EntityRecordByte(target, field.RecordOffset))
		} else {
			out = appendUint16LE(out, state.EntityRecordUint16(target, field.RecordOffset))
		}
	}
	return out
}

func serializeSVCDeltaPacketEntities(
	st *state.Packet,
	baseEntities []state.EntityRecord,
	packetEntities []state.EntityRecord,
) []byte {
	out := make([]byte, 0, len(packetEntities)*4+2)
	baseIndex, packetIndex := 0, 0

	for baseIndex < len(baseEntities) || packetIndex < len(packetEntities) {
		if packetIndex == len(packetEntities) ||
			(baseIndex < len(baseEntities) &&
				state.EntityNumber(baseEntities[baseIndex]) < state.EntityNumber(packetEntities[packetIndex])) {
			entityNumber := state.EntityNumber(baseEntities[baseIndex])
			out = appendUint16LE(out, entityNumber|protocol.URemove)
			baseIndex++
			continue
		}

		target := packetEntities[packetIndex]
		entityNumber := state.EntityNumber(target)
		var base state.EntityRecord
		hadBase := baseIndex < len(baseEntities) &&
			state.EntityNumber(baseEntities[baseIndex]) == entityNumber
		if hadBase {
			base = baseEntities[baseIndex]
			baseIndex++
		} else {
			base = entityBaseline(st, entityNumber)
			state.SetEntityNumber(&base, entityNumber)
		}
		packetIndex++

		if hadBase && entityWireEqual(base, target) {
			continue
		}
		out = appendPacketEntityUpdate(
			out,
			entityNumber,
			packetEntityWireBits(base, target),
			target,
		)
	}

	return append(out, 0, 0)
}

func entityBaseline(st *state.Packet, entityNumber uint16) state.EntityRecord {
	if record, ok := st.Baselines[entityNumber]; ok {
		return record
	}
	var record state.EntityRecord
	state.SetEntityNumber(&record, entityNumber)
	return record
}
