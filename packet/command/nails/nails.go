package nails

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Count   byte
	Command []Nail
}

type Nail struct {
	Bits []byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCNails)
	buf.PutByte(cmd.Count)

	for i := 0; i < len(cmd.Command); i++ {
		buf.PutBytes(cmd.Command[i].Bits)
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
		var nail Nail

		if nail.Bits, err = buf.GetBytes(6); err != nil {
			return nil, err
		}

		cmd.Command = append(cmd.Command, nail)
	}

	return &cmd, nil
}
