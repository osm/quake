package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/mvd"
)

type HiddenCommand struct {
	Size    uint32
	Type    uint16
	Command command.Command
}

func (cmd *HiddenCommand) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint32(cmd.Size)
	buf.PutUint16(cmd.Type)
	buf.PutBytes(cmd.Command.Bytes())

	return buf.Bytes()
}

func parseHiddenCommands(ctx *context.Context, data []byte) ([]*HiddenCommand, error) {
	var err error
	var cmds []*HiddenCommand

	buf := buffer.New(buffer.WithData(data))

	for buf.Off() < buf.Len() {
		var cmd HiddenCommand

		if cmd.Size, err = buf.GetUint32(); err != nil {
			return nil, err
		}

		if cmd.Type, err = buf.GetUint16(); err != nil {
			return nil, err
		}

		var c command.Command
		switch protocol.CommandType(cmd.Type) {
		case mvd.HiddenUserCommand:
			c, err = parseUserCommand(ctx, buf, cmd.Size)
		case mvd.HiddenUserCommandWeapon:
			c, err = parseWeapon(ctx, buf, cmd.Size)
		case mvd.HiddenDemoInfo:
			c, err = parseDemoInfo(ctx, buf, cmd.Size)
		case mvd.HiddenDamangeDone:
			c, err = parseDamageDone(ctx, buf, cmd.Size)
		case mvd.HiddenUserCommandWeaponServerSide:
			c, err = parseWeaponServerSide(ctx, buf, cmd.Size)
		case mvd.HiddenUserCommandWeaponInstruction:
			c, err = parseWeaponInstruction(ctx, buf, cmd.Size)
		default:
			c, err = parseUnknown(ctx, buf, cmd.Size)
		}

		if err != nil {
			return cmds, err
		}
		cmd.Command = c

		cmds = append(cmds, &cmd)
	}

	return cmds, nil
}
