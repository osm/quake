package setview

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	ViewEntity uint16
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCSetView)
	buf.PutUint16(cmd.ViewEntity)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.ViewEntity, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
