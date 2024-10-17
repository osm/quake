package updatepl

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	PlayerIndex byte
	PL          byte
}

func (cmd *Command) Bytes() []byte {
	return []byte{protocol.SVCUpdatePL, cmd.PlayerIndex, cmd.PL}
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.PL, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
