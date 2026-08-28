package qizmoblock

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol/qizmo"
)

type Command struct {
	Data []byte
}

func (cmd *Command) Bytes() []byte {
	return append([]byte{qizmo.SVCBlock}, cmd.Data...)
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	data, err := buf.GetBytes(qizmo.SVCBlockPayloadSize)
	if err != nil {
		return nil, err
	}
	return &Command{Data: data}, nil
}
