package updatecolors

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	PlayerIndex byte
	Color       byte
}

func (cmd *Command) Bytes() []byte {
	return []byte{protocol.SVCUpdateColors, cmd.PlayerIndex, cmd.Color}
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Color, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
