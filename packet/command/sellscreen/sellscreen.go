package sellscreen

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct{}

func (cmd *Command) Bytes() []byte {
	return []byte{protocol.SVCSellScreen}
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	return &Command{}, nil
}
