package fastupdate

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Bits     byte
	MoreBits byte
	Entity8  byte
	Entity16 uint16
	Model    byte
	Frame    byte
	ColorMap byte
	Skin     byte
	Effects  byte
	Origin   [3]float32
	Angle    [3]float32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(cmd.Bits)

	bits16 := uint16(cmd.Bits)
	bits16 &= 127

	if bits16&protocol.NQUMoreBits != 0 {
		buf.PutByte(cmd.MoreBits)
		bits16 |= uint16(cmd.MoreBits) << 8
	}

	if bits16&protocol.NQULongEntity != 0 {
		buf.PutUint16(cmd.Entity16)
	} else {
		buf.PutByte(cmd.Entity8)
	}

	if bits16&protocol.NQUModel != 0 {
		buf.PutByte(cmd.Model)
	}

	if bits16&protocol.NQUFrame != 0 {
		buf.PutByte(cmd.Frame)
	}

	if bits16&protocol.NQUColorMap != 0 {
		buf.PutByte(cmd.ColorMap)
	}

	if bits16&protocol.NQUSkin != 0 {
		buf.PutByte(cmd.Skin)
	}

	if bits16&protocol.NQUEffects != 0 {
		buf.PutByte(cmd.Effects)
	}

	if bits16&protocol.NQUOrigin1 != 0 {
		buf.PutCoord16(cmd.Origin[0])
	}

	if bits16&protocol.NQUAngle1 != 0 {
		buf.PutAngle8(cmd.Angle[0])
	}

	if bits16&protocol.NQUOrigin2 != 0 {
		buf.PutCoord16(cmd.Origin[1])
	}

	if bits16&protocol.NQUAngle2 != 0 {
		buf.PutAngle8(cmd.Angle[1])
	}

	if bits16&protocol.NQUOrigin3 != 0 {
		buf.PutCoord16(cmd.Origin[2])
	}

	if bits16&protocol.NQUAngle3 != 0 {
		buf.PutAngle8(cmd.Angle[2])
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer, bits byte) (*Command, error) {
	var err error
	var cmd Command

	cmd.Bits = bits

	bits16 := uint16(bits)
	bits16 &= 127

	if bits16&protocol.NQUMoreBits != 0 {
		if cmd.MoreBits, err = buf.ReadByte(); err != nil {
			return nil, err
		}
		bits16 |= uint16(cmd.MoreBits) << 8
	}

	if bits16&protocol.NQULongEntity != 0 {
		if cmd.Entity16, err = buf.GetUint16(); err != nil {
			return nil, err
		}
	} else {
		if cmd.Entity8, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUModel != 0 {
		if cmd.Model, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUFrame != 0 {
		if cmd.Frame, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUColorMap != 0 {
		if cmd.ColorMap, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUSkin != 0 {
		if cmd.Skin, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUEffects != 0 {
		if cmd.Effects, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUOrigin1 != 0 {
		if cmd.Origin[0], err = buf.GetCoord16(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUAngle1 != 0 {
		if cmd.Angle[0], err = buf.GetAngle8(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUOrigin2 != 0 {
		if cmd.Origin[1], err = buf.GetCoord16(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUAngle2 != 0 {
		if cmd.Angle[1], err = buf.GetAngle8(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUOrigin3 != 0 {
		if cmd.Origin[2], err = buf.GetCoord16(); err != nil {
			return nil, err
		}
	}

	if bits16&protocol.NQUAngle3 != 0 {
		if cmd.Angle[2], err = buf.GetAngle8(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
