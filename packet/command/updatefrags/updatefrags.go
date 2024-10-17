package updatefrags

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	PlayerIndex byte
	Frags       int16
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCUpdateFrags)
	buf.PutByte(cmd.PlayerIndex)
	buf.PutInt16(cmd.Frags)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Frags, err = buf.GetInt16(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
