package clc

import (
	"errors"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/bad"
	"github.com/osm/quake/packet/command/delta"
	"github.com/osm/quake/packet/command/ftevoicechatc"
	"github.com/osm/quake/packet/command/move"
	"github.com/osm/quake/packet/command/mvdweapon"
	"github.com/osm/quake/packet/command/nopc"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/command/tmove"
	"github.com/osm/quake/packet/command/upload"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
	"github.com/osm/quake/protocol/mvd"
)

var ErrUnknownCommandType = errors.New("unknown command type")

type GameData struct {
	Seq      uint32
	Ack      uint32
	QPort    uint16
	Commands []command.Command
}

func (gd *GameData) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint32(gd.Seq)
	buf.PutUint32(gd.Ack)
	buf.PutUint16(gd.QPort)

	for _, c := range gd.Commands {
		buf.PutBytes(c.Bytes())
	}

	return buf.Bytes()
}

func parseGameData(ctx *context.Context, buf *buffer.Buffer) (*GameData, error) {
	var err error
	var pkg GameData

	if pkg.Seq, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if pkg.Ack, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if pkg.QPort, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	var cmd command.Command
	for buf.Off() < buf.Len() {
		typ, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}

		switch protocol.CommandType(typ) {
		case protocol.CLCBad:
			cmd, err = bad.Parse(ctx, buf, protocol.CLCBad)
		case protocol.CLCNOP:
			cmd, err = nopc.Parse(ctx, buf)
		case protocol.CLCDoubleMove:
		case protocol.CLCMove:
			cmd, err = move.Parse(ctx, buf)
		case protocol.CLCStringCmd:
			cmd, err = stringcmd.Parse(ctx, buf)
		case protocol.CLCDelta:
			cmd, err = delta.Parse(ctx, buf)
		case protocol.CLCTMove:
			cmd, err = tmove.Parse(ctx, buf)
		case protocol.CLCUpload:
			cmd, err = upload.Parse(ctx, buf)
		case fte.CLCVoiceChat:
			cmd, err = ftevoicechatc.Parse(ctx, buf)
		case mvd.CLCWeapon:
			cmd, err = mvdweapon.Parse(ctx, buf)
		default:
			return nil, ErrUnknownCommandType
		}

		if err != nil {
			return nil, err
		}

		pkg.Commands = append(pkg.Commands, cmd)
	}

	return &pkg, nil
}
