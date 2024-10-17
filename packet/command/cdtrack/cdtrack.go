package cdtrack

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	IsNQ bool

	Track byte
	Loop  byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCCDTrack)
	buf.PutByte(cmd.Track)

	if cmd.IsNQ {
		buf.PutByte(cmd.Loop)
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.IsNQ = ctx.GetIsNQ()

	if cmd.Track, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.IsNQ {
		if cmd.Loop, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
