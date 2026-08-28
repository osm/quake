package standard

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/state"
)

func (p *packetParser) parsePacketEntities(base map[uint16]state.EntityRecord) error {
	entities, err := parsePacketEntityRecords(p.state, p.reader, base)
	if err != nil {
		return err
	}
	p.packetEntities = entities
	return nil
}

func ParsePacketEntities(
	packetState *state.Packet,
	data []byte,
	base map[uint16]state.EntityRecord,
) (map[uint16]state.EntityRecord, int, error) {
	reader := newReader(data)
	entities, err := parsePacketEntityRecords(packetState, reader, base)
	return entities, len(data) - reader.Len(), err
}

func parsePacketEntityRecords(
	packetState *state.Packet,
	reader *reader,
	base map[uint16]state.EntityRecord,
) (map[uint16]state.EntityRecord, error) {
	entities := state.CloneEntityMap(base)
	for {
		header, err := reader.readUint16()
		if err != nil {
			return nil, err
		}
		if header == 0 {
			return entities, nil
		}

		entityNumber := header & protocol.UCheckMoreBits
		bits := header &^ protocol.UCheckMoreBits
		if bits&protocol.UMoreBits != 0 {
			lo, err := reader.ReadByte()
			if err != nil {
				return nil, err
			}
			bits |= uint16(lo)
		}
		if bits&protocol.URemove != 0 {
			delete(entities, entityNumber)
			continue
		}

		record, ok := entities[entityNumber]
		if !ok {
			record = packetState.Baselines[entityNumber]
			state.SetEntityNumber(&record, entityNumber)
		}
		clear(record[wire.EntityOriginCarryOffset:wire.EntityCarryEnd])

		for _, field := range wire.PacketEntityFields {
			if bits&field.Mask == 0 {
				continue
			}
			if field.Size == 1 {
				value, err := reader.ReadByte()
				if err != nil {
					return nil, err
				}
				state.SetEntityRecordByte(&record, field.RecordOffset, value)
				continue
			}
			value, err := reader.readUint16()
			if err != nil {
				return nil, err
			}
			state.SetEntityRecordUint16(&record, field.RecordOffset, value)
		}

		state.SetEntityMask(&record, bits)
		entities[entityNumber] = record
	}
}
