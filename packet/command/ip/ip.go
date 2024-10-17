package ip

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type Command struct {
	String string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutString(cmd.String)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	return &Command{}, nil
}
