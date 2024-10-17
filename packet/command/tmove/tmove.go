package tmove

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Coord [3]uint16
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.CLCTMove)

	for i := 0; i < 3; i++ {
		buf.PutUint16(cmd.Coord[i])
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	for i := 0; i < 3; i++ {
		if cmd.Coord[i], err = buf.GetUint16(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
