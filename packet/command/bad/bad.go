package bad

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Type protocol.CommandType
}

func (cmd *Command) Bytes() []byte {
	return []byte{byte(cmd.Type)}
}

func Parse(ctx *context.Context, buf *buffer.Buffer, typ protocol.CommandType) (*Command, error) {
	return &Command{Type: typ}, nil
}
