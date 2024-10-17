package getchallenge

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type Command struct{}

func (cmd *Command) Bytes() []byte {
	return []byte("getchallenge\n")
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	return &Command{}, nil
}
