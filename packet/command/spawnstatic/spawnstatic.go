package spawnstatic

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/baseline"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Baseline *baseline.Command
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCSpawnStatic)

	if cmd.Baseline != nil {
		buf.PutBytes(cmd.Baseline.Bytes())
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Baseline, err = baseline.Parse(ctx, buf); err != nil {
		return nil, err
	}

	return &cmd, nil
}
