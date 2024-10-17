package chokecount

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Count byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCChokeCount)
	buf.PutByte(cmd.Count)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Count, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
