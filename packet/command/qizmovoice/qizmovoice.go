package qizmovoice

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Data []byte
}

func (cmd *Command) Bytes() []byte {
	return append([]byte{
		protocol.SVCQizmoVoice},
		cmd.Data...,
	)
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Data, err = buf.GetBytes(34); err != nil {
		return nil, err
	}

	return &cmd, nil
}
