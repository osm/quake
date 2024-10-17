package setinfo

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	PlayerIndex byte
	Key         string
	Value       string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCSetInfo)
	buf.PutByte(cmd.PlayerIndex)
	buf.PutString(cmd.Key)
	buf.PutString(cmd.Value)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Key, err = buf.GetString(); err != nil {
		return nil, err
	}

	if cmd.Value, err = buf.GetString(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
