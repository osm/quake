package updateentertime

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	PlayerIndex byte
	EnterTime   float32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCUpdateEnterTime)
	buf.PutByte(cmd.PlayerIndex)
	buf.PutFloat32(cmd.EnterTime)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.EnterTime, err = buf.GetFloat32(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
