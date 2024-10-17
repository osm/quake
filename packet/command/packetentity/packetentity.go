package packetentity

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/packetentitydelta"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
)

type Command struct {
	FTEProtocolExtension uint32

	Bits              uint16
	MoreBits          byte
	EvenMoreBits      byte
	PacketEntityDelta *packetentitydelta.Command
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint16(cmd.Bits)

	if cmd.Bits == 0 {
		goto end
	}

	if cmd.Bits&protocol.URemove != 0 {
		if cmd.Bits&protocol.UMoreBits != 0 &&
			cmd.FTEProtocolExtension&fte.ExtensionEntityDbl != 0 {
			buf.PutByte(cmd.MoreBits)

			if cmd.MoreBits&fte.UEvenMore != 0 {
				buf.PutByte(cmd.EvenMoreBits)
			}
		}
		goto end
	}

	if cmd.PacketEntityDelta != nil {
		buf.PutBytes(cmd.PacketEntityDelta.Bytes())
	}

end:
	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) ([]*Command, error) {
	var err error
	var cmds []*Command

	for {
		var cmd Command
		cmd.FTEProtocolExtension = ctx.GetFTEProtocolExtension()

		if cmd.Bits, err = buf.GetUint16(); err != nil {
			return nil, err
		}

		if cmd.Bits == 0 {
			break
		}

		if cmd.Bits&protocol.URemove != 0 {
			if cmd.Bits&protocol.UMoreBits != 0 &&
				cmd.FTEProtocolExtension&fte.ExtensionEntityDbl != 0 {
				if cmd.MoreBits, err = buf.ReadByte(); err != nil {
					return nil, err
				}

				if cmd.MoreBits&fte.UEvenMore != 0 {
					if cmd.EvenMoreBits, err = buf.ReadByte(); err != nil {
						return nil, err
					}
				}
			}
			goto next
		}

		cmd.PacketEntityDelta, err = packetentitydelta.Parse(ctx, buf, cmd.Bits)
		if err != nil {
			return nil, err
		}
	next:
		cmds = append(cmds, &cmd)
	}

	return cmds, nil
}
