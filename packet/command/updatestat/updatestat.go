package updatestat

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	IsNQ bool

	Stat    byte
	Value8  byte
	Value32 uint32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCUpdateStat)
	buf.PutByte(cmd.Stat)

	if cmd.IsNQ {
		buf.PutUint32(cmd.Value32)
	} else {
		buf.PutByte(cmd.Value8)
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.IsNQ = ctx.GetIsNQ()

	if cmd.Stat, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.IsNQ {
		if cmd.Value32, err = buf.GetUint32(); err != nil {
			return nil, err
		}
	} else {
		if cmd.Value8, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
