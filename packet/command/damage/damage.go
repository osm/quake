package damage

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	CoordSize uint8

	Armor byte
	Blood byte
	Coord [3]float32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	writeCoord := buf.PutCoord16
	if cmd.CoordSize == 4 {
		writeCoord = buf.PutCoord32
	}

	buf.PutByte(byte(protocol.SVCDamage))
	buf.PutByte(cmd.Armor)
	buf.PutByte(cmd.Blood)

	for i := 0; i < 3; i++ {
		writeCoord(cmd.Coord[i])
	}

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

	if cmd.Armor, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Blood, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		if cmd.Coord[i], err = readCoord(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
