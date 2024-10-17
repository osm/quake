package mvdweapon

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol/mvd"
)

type Command struct {
	Bits    byte
	Age     byte
	Weapons []byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(mvd.CLCWeapon)
	buf.PutByte(cmd.Bits)
	buf.PutByte(cmd.Age)
	buf.PutBytes(cmd.Weapons)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Bits, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Bits&mvd.CLCWeaponForgetRanking != 0 {
		if cmd.Age, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	for {
		weapon, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}

		if weapon == 0 {
			break
		}

		cmd.Weapons = append(cmd.Weapons, weapon)
	}

	return &cmd, nil
}
