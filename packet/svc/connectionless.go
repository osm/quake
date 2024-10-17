package svc

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/a2aping"
	"github.com/osm/quake/packet/command/a2cclientcommand"
	"github.com/osm/quake/packet/command/a2cprint"
	"github.com/osm/quake/packet/command/disconnect"
	"github.com/osm/quake/packet/command/passthrough"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/command/s2cconnection"
	"github.com/osm/quake/protocol"
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

	typ, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	var cmd command.Command
	switch protocol.CommandType(typ) {
	case protocol.S2CConnection:
		cmd, err = s2cconnection.Parse(ctx, buf)
	case protocol.A2CClientCommand:
		cmd, err = a2cclientcommand.Parse(ctx, buf)
	case protocol.A2CPrint:
		cmd, err = a2cprint.Parse(ctx, buf)
	case protocol.A2APing:
		cmd, err = a2aping.Parse(ctx, buf)
	case protocol.S2CChallenge:
		cmd, err = s2cchallenge.Parse(ctx, buf)
	case protocol.SVCDisconnect:
		cmd, err = disconnect.Parse(ctx, buf)
	default:
		cmd, err = passthrough.Parse(ctx, buf, "")
	}

	if err != nil {
		return nil, err
	}
	pkg.Command = cmd

	return &pkg, nil
}
