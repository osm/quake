package ftedownload

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/download"
)

type Command struct {
	Number  int32
	Command byte
	Chunk   *download.Command
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutString("\\chunk")
	buf.PutInt32(cmd.Number)
	buf.PutByte(cmd.Command)
	if cmd.Chunk != nil {
		buf.PutBytes(cmd.Chunk.Bytes())
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if err := buf.Skip(6); err != nil {
		return nil, err
	}

	if cmd.Number, err = buf.GetInt32(); err != nil {
		return nil, err
	}

	if cmd.Command, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Chunk, err = download.Parse(ctx, buf); err != nil {
		return nil, err
	}

	return nil, nil
}
