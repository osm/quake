package nails2

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Count byte
	Nails []Nail2
}

type Nail2 struct {
	Index byte
	Bits  []byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCNails2)
	buf.PutByte(cmd.Count)

	for i := 0; i < len(cmd.Nails); i++ {
		buf.PutByte(cmd.Nails[i].Index)
		buf.PutBytes(cmd.Nails[i].Bits)
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Count, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	for i := 0; i < int(cmd.Count); i++ {
		var nail Nail2

		if nail.Index, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		if nail.Bits, err = buf.GetBytes(6); err != nil {
			return nil, err
		}

		cmd.Nails = append(cmd.Nails, nail)
	}

	return &cmd, nil
}
