package ftespawnbaseline

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/packetentitydelta"
	"github.com/osm/quake/protocol/fte"
)

type Command struct {
	Index uint16
	Delta *packetentitydelta.Command
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(fte.SVCSpawnBaseline)
	buf.PutUint16(cmd.Index)

	if cmd.Delta != nil {
		buf.PutBytes(cmd.Delta.Bytes())
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Index, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Delta, err = packetentitydelta.Parse(ctx, buf, cmd.Index); err != nil {
		return nil, err
	}

	return &cmd, nil
}
