package spawnstaticsound

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	CoordSize uint8

	Coord       [3]float32
	SoundIndex  byte
	Volume      byte
	Attenuation byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	writeCoord := buf.PutCoord16
	if cmd.CoordSize == 4 {
		writeCoord = buf.PutCoord32
	}

	buf.PutByte(protocol.SVCSpawnStaticSound)

	for i := 0; i < 3; i++ {
		writeCoord(cmd.Coord[i])
	}

	buf.PutByte(cmd.SoundIndex)
	buf.PutByte(cmd.Volume)
	buf.PutByte(cmd.Attenuation)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.CoordSize = ctx.GetCoordSize()
	readCoord := buf.GetCoord16
	if cmd.CoordSize == 4 {
		readCoord = buf.GetCoord32
	}

	for i := 0; i < 3; i++ {
		if cmd.Coord[i], err = readCoord(); err != nil {
			return nil, err
		}
	}

	if cmd.SoundIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Volume, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Attenuation, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
