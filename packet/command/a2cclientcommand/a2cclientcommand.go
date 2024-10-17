package a2cclientcommand

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Command string
	LocalID string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.A2CClientCommand)
	buf.PutString(cmd.Command)
	buf.PutString(cmd.LocalID)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Command, err = buf.GetString(); err != nil {
		return nil, err
	}

	if cmd.LocalID, err = buf.GetString(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
