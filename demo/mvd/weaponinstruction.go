package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type WeaponInstruction struct {
	PlayerIndex byte
	Bits        byte
	Seq         uint32
	Mode        uint32
	WeaponList  []byte
}

func (cmd *WeaponInstruction) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(cmd.PlayerIndex)
	buf.PutByte(cmd.Bits)
	buf.PutUint32(cmd.Seq)
	buf.PutUint32(cmd.Mode)
	buf.PutBytes(cmd.WeaponList)

	return buf.Bytes()
}

func parseWeaponInstruction(
	ctx *context.Context,
	buf *buffer.Buffer,
	size uint32,
) (*WeaponInstruction, error) {
	var err error
	var cmd WeaponInstruction

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Bits, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Seq, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if cmd.Mode, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if cmd.WeaponList, err = buf.GetBytes(10); err != nil {
		return nil, err
	}

	return &cmd, nil
}
