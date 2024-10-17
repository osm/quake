package connect

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/osm/quake/common/args"
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Command     string
	Version     string
	QPort       uint16
	ChallengeID string
	UserInfo    *infostring.InfoString
	Extensions  []*protocol.Extension
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutBytes([]byte(cmd.Command + " "))
	buf.PutBytes([]byte(cmd.Version + " "))
	buf.PutBytes([]byte(fmt.Sprintf("%v ", cmd.QPort)))
	buf.PutBytes([]byte(cmd.ChallengeID + " "))

	if cmd.UserInfo != nil {
		buf.PutBytes(cmd.UserInfo.Bytes())
	}

	buf.PutByte(0x0a)

	for _, ext := range cmd.Extensions {
		if ext != nil {
			buf.PutBytes(ext.Bytes())
		}
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer, arg args.Arg) (*Command, error) {
	var cmd Command

	if len(arg.Args) < 4 {
		return nil, errors.New("unexpected length of args")
	}

	cmd.Command = arg.Cmd
	cmd.Version = arg.Args[0]

	qPort, err := strconv.ParseUint(arg.Args[1], 10, 16)
	if err != nil {
		return nil, err
	}
	cmd.QPort = uint16(qPort)

	cmd.ChallengeID = arg.Args[2]
	cmd.UserInfo = infostring.Parse(arg.Args[3])

	return &cmd, nil
}
