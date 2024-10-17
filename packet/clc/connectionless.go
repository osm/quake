package clc

import (
	"errors"

	"github.com/osm/quake/common/args"
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/getchallenge"
	"github.com/osm/quake/packet/command/passthrough"
)

type Connectionless struct {
	Command command.Command
}

func (cmd *Connectionless) Bytes() []byte {
	buf := buffer.New()

	buf.PutInt32(-1)
	buf.PutBytes(cmd.Command.Bytes())

	return buf.Bytes()
}

func parseConnectionless(ctx *context.Context, buf *buffer.Buffer) (*Connectionless, error) {
	var err error
	var pkg Connectionless

	if err := buf.Skip(4); err != nil {
		return nil, err
	}

	var str string
	if str, err = buf.GetString(); err != nil {
		return nil, err
	}

	args := args.Parse(str)
	if len(args) != 1 {
		return nil, errors.New("unexpected length of parsed arguments")
	}

	arg := args[0]

	var cmd command.Command
	switch arg.Cmd {
	case "connect":
		cmd, err = connect.Parse(ctx, buf, arg)
	case "getchallenge":
		cmd, err = getchallenge.Parse(ctx, buf)
	default:
		cmd, err = passthrough.Parse(ctx, buf, str)
	}

	if err != nil {
		return nil, err
	}
	pkg.Command = cmd

	return &pkg, nil
}
