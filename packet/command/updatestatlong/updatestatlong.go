package updatestatlong

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Stat  byte
	Value int32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCUpdateStatLong)
	buf.PutByte(cmd.Stat)
	buf.PutInt32(cmd.Value)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Stat, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Value, err = buf.GetInt32(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
