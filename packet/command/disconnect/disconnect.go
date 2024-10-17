package disconnect

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	IsMVD bool
	IsQWD bool

	String string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCDisconnect)

	if cmd.IsMVD || cmd.IsQWD {
		buf.PutString(cmd.String)
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.IsQWD = ctx.GetIsQWD()
	cmd.IsMVD = ctx.GetIsMVD()

	if cmd.IsQWD || cmd.IsMVD {
		if cmd.String, err = buf.GetString(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
