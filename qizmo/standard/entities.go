package standard

import (
	"encoding/binary"

	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/state"
)

const (
	spawnBaselinePayloadSize = 13
	spawnBaselineAxisSize    = 3
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
	byteOffsets := [...]int{
		wire.EntityModelOffset,
		wire.EntityFrameOffset,
		wire.EntityColorMapOffset,
		wire.EntitySkinNumOffset,
	}
	for i, offset := range byteOffsets {
		state.SetEntityRecordByte(&record, offset, payload[i])
	}
	var origin [3]uint16
	for axis := range 3 {
		baseOffset := len(byteOffsets) + axis*spawnBaselineAxisSize
		origin[axis] = binary.LittleEndian.Uint16(payload[baseOffset:])
		state.SetEntityRecordByte(&record, wire.EntityAngleOffset+axis, payload[baseOffset+2])
	}
	state.SetEntityOrigin(&record, origin)

	p.state.Baselines[entityNumber] = record
	return nil
}
