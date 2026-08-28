package standard

import (
	"encoding/binary"

	"github.com/osm/quake/qizmo/state"
)

const (
	spawnBaselinePayloadSize = 13
)

func (p *packetParser) parseSpawnBaseline() error {
	entityNumber, err := p.reader.readUint16()
	if err != nil {
		return err
	}
	payload, err := p.reader.readN(spawnBaselinePayloadSize)
	if err != nil {
		return err
	}

	var record state.EntityRecord
	state.SetEntityNumber(&record, entityNumber)
	state.SetEntityRecordByte(&record, 4, payload[0])
	state.SetEntityRecordByte(&record, 5, payload[1])
	state.SetEntityRecordByte(&record, 6, payload[2])
	state.SetEntityRecordByte(&record, 7, payload[3])
	state.SetEntityRecordByte(&record, 8, 0)
	state.SetEntityOrigin(&record, [3]uint16{
		binary.LittleEndian.Uint16(payload[4:6]),
		binary.LittleEndian.Uint16(payload[7:9]),
		binary.LittleEndian.Uint16(payload[10:12]),
	})
	state.SetEntityRecordByte(&record, 9, payload[6])
	state.SetEntityRecordByte(&record, 10, payload[9])
	state.SetEntityRecordByte(&record, 11, payload[12])

	p.state.Baselines[entityNumber] = record
	return nil
}
