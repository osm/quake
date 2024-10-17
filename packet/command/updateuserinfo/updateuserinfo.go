package updateuserinfo

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	PlayerIndex byte
	UserID      uint32
	UserInfo    string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCUpdateUserInfo)
	buf.PutByte(cmd.PlayerIndex)
	buf.PutUint32(cmd.UserID)
	buf.PutString(cmd.UserInfo)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.UserID, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if cmd.UserInfo, err = buf.GetString(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
