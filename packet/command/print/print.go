package print

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	IsNQ bool

	ID     byte
	String string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCPrint)

	if !cmd.IsNQ {
		buf.PutByte(cmd.ID)
	}

	buf.PutString(cmd.String)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.IsNQ = ctx.GetIsNQ()

	if !cmd.IsNQ {
		if cmd.ID, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.String, err = buf.GetString(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
