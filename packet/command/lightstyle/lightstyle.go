package lightstyle

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Index   byte
	Command string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCLightStyle)
	buf.PutByte(cmd.Index)
	buf.PutString(cmd.Command)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Index, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Command, err = buf.GetString(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
