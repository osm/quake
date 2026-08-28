package qizmostring

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol/qizmo"
)

type Command struct {
	String string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()
	buf.PutByte(qizmo.SVCString)
	buf.PutString(cmd.String)
	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	value, err := buf.GetString()
	if err != nil {
		return nil, err
	}
	return &Command{String: value}, nil
}
