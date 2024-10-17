package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type WeaponServerSide struct {
	PlayerIndex byte
	Items       uint32
	Shells      byte
	Nails       byte
	Rockets     byte
	Cells       byte
	Choice      byte
	String      string
}

func (cmd *WeaponServerSide) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(cmd.PlayerIndex)
	buf.PutUint32(cmd.Items)
	buf.PutByte(cmd.Shells)
	buf.PutByte(cmd.Nails)
	buf.PutByte(cmd.Rockets)
	buf.PutByte(cmd.Cells)
	buf.PutByte(cmd.Choice)
	buf.PutString(cmd.String)

	return buf.Bytes()
}

func parseWeaponServerSide(
	ctx *context.Context,
	buf *buffer.Buffer,
	size uint32,
) (*WeaponServerSide, error) {
	var err error
	var cmd WeaponServerSide

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Items, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if cmd.Shells, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Nails, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Rockets, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Cells, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Choice, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.String, err = buf.GetString(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
