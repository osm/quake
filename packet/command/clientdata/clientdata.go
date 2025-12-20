package clientdata

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Bits         uint16
	ViewHeight   byte
	IdealPitch   byte
	PunchAngle   [3]byte
	Velocity     [3]byte
	Items        uint32
	WeaponFrame  byte
	Armor        byte
	Weapon       byte
	Health       uint16
	ActiveAmmo   byte
	AmmoShells   byte
	AmmoNails    byte
	AmmoRockets  byte
	AmmoCells    byte
	ActiveWeapon byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCClientData)
	buf.PutUint16(cmd.Bits)

	if cmd.Bits&protocol.SUViewHeight != 0 {
		buf.PutByte(cmd.ViewHeight)
	}

	if cmd.Bits&protocol.SUIdealPitch != 0 {
		buf.PutByte(cmd.IdealPitch)
	}

	for i := 0; i < 3; i++ {
		if cmd.Bits&(protocol.SUPunch1<<i) != 0 {
			buf.PutByte(cmd.PunchAngle[i])
		}

		if cmd.Bits&(protocol.SUVelocity1<<i) != 0 {
			buf.PutByte(cmd.Velocity[i])
		}
	}

	if cmd.Bits&protocol.SUItems != 0 {
		buf.PutUint32(cmd.Items)
	}

	if cmd.Bits&protocol.SUWeaponFrame != 0 {
		buf.PutByte(cmd.WeaponFrame)
	}

	if cmd.Bits&protocol.SUArmor != 0 {
		buf.PutByte(cmd.Armor)
	}

	if cmd.Bits&protocol.SUWeapon != 0 {
		buf.PutByte(cmd.Weapon)
	}

	buf.PutUint16(cmd.Health)
	buf.PutByte(cmd.ActiveAmmo)
	buf.PutByte(cmd.AmmoShells)
	buf.PutByte(cmd.AmmoNails)
	buf.PutByte(cmd.AmmoRockets)
	buf.PutByte(cmd.AmmoCells)
	buf.PutByte(cmd.ActiveWeapon)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Bits, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Bits&protocol.SUViewHeight != 0 {
		if cmd.ViewHeight, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.SUIdealPitch != 0 {
		if cmd.IdealPitch, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	for i := 0; i < 3; i++ {
		if cmd.Bits&(protocol.SUPunch1<<i) != 0 {
			if cmd.PunchAngle[i], err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}

		if cmd.Bits&(protocol.SUVelocity1<<i) != 0 {
			if cmd.Velocity[i], err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}
	}

	if cmd.Items, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if cmd.Bits&protocol.SUWeaponFrame != 0 {
		if cmd.WeaponFrame, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.SUArmor != 0 {
		if cmd.Armor, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.SUWeapon != 0 {
		if cmd.Weapon, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.Health, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.ActiveAmmo, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.AmmoShells, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.AmmoNails, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.AmmoRockets, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.AmmoCells, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.ActiveWeapon, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
