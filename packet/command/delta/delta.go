package delta

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Seq byte
}

func (cmd *Command) Bytes() []byte {
	return []byte{protocol.CLCDelta, cmd.Seq}
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Seq, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}