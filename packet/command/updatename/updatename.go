package updatename

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	PlayerIndex byte
	Name        string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCUpdateName)
	buf.PutByte(cmd.PlayerIndex)
	buf.PutString(cmd.Name)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Name, err = buf.GetString(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
