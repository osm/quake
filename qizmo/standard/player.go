package standard

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/state"
)

func (p *packetParser) parsePlayerInfo() error {
	player, err := p.reader.ReadByte()
	if err != nil {
		return err
	}
	flags, err := p.reader.readUint16()
	if err != nil {
		return err
	}
	positionAndFrame, err := p.reader.readN(7)
	if err != nil {
		return err
	}

	record := state.PlayerRecordBytesLE(p.state.DefaultPlayer())
	record[wire.PlayerIndexOffset] = player
	copy(record[wire.PlayerOriginOffset:wire.PlayerIndexOffset], positionAndFrame)
	if flags&protocol.PFDead != 0 {
		record[wire.PlayerMotionMaskOffset] = wire.PlayerDead
	}

	if flags&protocol.PFMsec != 0 {
		msec, err := p.reader.ReadByte()
		if err != nil {
			return err
		}
		record[wire.PlayerMsecOffset] = msec
	}

	if flags&protocol.PFCommand != 0 {
		extra, err := p.reader.ReadByte()
		if err != nil {
			return err
		}
		for _, field := range wire.PlayerCommandFields {
			if extra&field.Mask == 0 {
				continue
			}
			value, err := p.reader.readN(field.Size)
			if err != nil {
				return err
			}
			if field.RecordOffset != wire.UntrackedRecordOffset {
				copy(record[field.RecordOffset:field.RecordOffset+field.Size], value)
			}
		}
		commandMsec, err := p.reader.ReadByte()
		if err != nil {
			return err
		}
		record[wire.PlayerCommandMsecOffset] = commandMsec
	}

	for _, field := range wire.PlayerVelocityFields {
		if flags&field.Mask == 0 {
			continue
		}
		value, err := p.reader.readN(field.Size)
		if err != nil {
			return err
		}
		copy(record[field.RecordOffset:field.RecordOffset+field.Size], value)
	}
	for _, field := range wire.PlayerByteFields {
		if flags&field.Mask == 0 {
			continue
		}
		value, err := p.reader.ReadByte()
		if err != nil {
			return err
		}
		record[field.RecordOffset] = value
	}

	p.state.CurrentPlayers = append(p.state.CurrentPlayers, state.PlayerRecordFromBytesLE(record))
	return nil
}
