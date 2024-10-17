package entgravity

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	EntGravity float32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCEntGravity)
	buf.PutFloat32(cmd.EntGravity)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.EntGravity, err = buf.GetFloat32(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
