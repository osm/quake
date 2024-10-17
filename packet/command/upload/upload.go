package upload

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Size    int16
	Percent byte
	Data    []byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.CLCUpload)
	buf.PutInt16(cmd.Size)
	buf.PutByte(cmd.Percent)
	buf.PutBytes(cmd.Data)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Size, err = buf.GetInt16(); err != nil {
		return nil, err
	}

	if cmd.Percent, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Data, err = buf.GetBytes(int(cmd.Size)); err != nil {
		return nil, err
	}

	return &cmd, nil
}
