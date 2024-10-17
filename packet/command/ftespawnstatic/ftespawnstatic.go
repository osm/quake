package ftespawnstatic

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/packetentitydelta"
	"github.com/osm/quake/protocol/fte"
)

type Command struct {
	Bits  uint16
	Delta *packetentitydelta.Command
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(fte.SVCSpawnStatic)
	buf.PutUint16(cmd.Bits)

	if cmd.Delta != nil {
		buf.PutBytes(cmd.Delta.Bytes())
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Bits, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Delta, err = packetentitydelta.Parse(ctx, buf, cmd.Bits); err != nil {
		return nil, err
	}

	return &cmd, nil
}
