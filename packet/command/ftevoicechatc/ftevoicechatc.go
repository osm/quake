package ftevoicechatc

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol/fte"
)

type Command struct {
	Gen  byte
	Seq  byte
	Size uint16
	Data []byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(fte.CLCVoiceChat)
	buf.PutByte(cmd.Gen)
	buf.PutByte(cmd.Seq)
	buf.PutUint16(cmd.Size)
	buf.PutBytes(cmd.Data)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Gen, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Seq, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Size, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Data, err = buf.GetBytes(int(cmd.Size)); err != nil {
		return nil, err
	}

	return &cmd, nil
}
