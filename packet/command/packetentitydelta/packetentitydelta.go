package packetentitydelta

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
	"github.com/osm/quake/protocol/mvd"
)

type Command struct {
	AngleSize            uint8
	CoordSize            uint8
	FTEProtocolExtension uint32
	MVDProtocolExtension uint32

	bits uint16

	MoreBits     byte
	EvenMoreBits byte
	YetMoreBits  byte

	ModelIndex byte
	Frame      byte
	ColorMap   byte
	Skin       byte
	Effects    byte
	Coord      [3]float32
	Angle      [3]float32
	Trans      byte
	ColorMod   [3]byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	writeAngle := buf.PutAngle8
	if cmd.AngleSize == 2 {
		writeAngle = buf.PutAngle16
	}

	writeCoord := buf.PutCoord16
	if cmd.CoordSize == 4 || cmd.MVDProtocolExtension&mvd.ExtensionFloatCoords != 0 {
		writeCoord = buf.PutCoord32
	}

	bits := cmd.bits
	bits &= ^uint16(511)

	if bits&protocol.UMoreBits != 0 {
		buf.PutByte(cmd.MoreBits)
		bits |= uint16(cmd.MoreBits)
	}

	var moreBits uint16
	if bits&fte.UEvenMore != 0 && cmd.FTEProtocolExtension > 0 {
		buf.PutByte(cmd.EvenMoreBits)
		moreBits = uint16(cmd.EvenMoreBits)

		if cmd.EvenMoreBits&fte.UYetMore != 0 {
			buf.PutByte(cmd.YetMoreBits)
			moreBits |= uint16(cmd.YetMoreBits) << 8
		}
	}

	if bits&protocol.UModel != 0 {
		buf.PutByte(cmd.ModelIndex)
	}

	if bits&protocol.UFrame != 0 {
		buf.PutByte(cmd.Frame)
	}

	if bits&protocol.UColorMap != 0 {
		buf.PutByte(cmd.ColorMap)
	}

	if bits&protocol.USkin != 0 {
		buf.PutByte(cmd.Skin)
	}

	if bits&protocol.UEffects != 0 {
		buf.PutByte(cmd.Effects)
	}

	if bits&protocol.UOrigin1 != 0 {
		writeCoord(cmd.Coord[0])
	}

	if bits&protocol.UAngle1 != 0 {
		writeAngle(cmd.Angle[0])
	}

	if bits&protocol.UOrigin2 != 0 {
		writeCoord(cmd.Coord[1])
	}

	if bits&protocol.UAngle2 != 0 {
		writeAngle(cmd.Angle[1])
	}

	if bits&protocol.UOrigin3 != 0 {
		writeCoord(cmd.Coord[2])
	}

	if bits&protocol.UAngle3 != 0 {
		writeAngle(cmd.Angle[2])
	}

	if moreBits&fte.UTrans != 0 && cmd.FTEProtocolExtension&fte.ExtensionTrans != 0 {
		buf.PutByte(cmd.Trans)
	}

	if moreBits&fte.UColorMod != 0 && cmd.FTEProtocolExtension&fte.ExtensionColorMod != 0 {
		for i := 0; i < len(cmd.ColorMod); i++ {
			buf.PutByte(cmd.ColorMod[i])
		}
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer, bits uint16) (*Command, error) {
	var err error
	var cmd Command

	cmd.FTEProtocolExtension = ctx.GetFTEProtocolExtension()
	cmd.MVDProtocolExtension = ctx.GetMVDProtocolExtension()

	cmd.AngleSize = ctx.GetAngleSize()
	readAngle := buf.GetAngle8
	if cmd.AngleSize == 2 {
		readAngle = buf.GetAngle16
	}

	cmd.CoordSize = ctx.GetCoordSize()
	readCoord := buf.GetCoord16
	if cmd.CoordSize == 4 || cmd.MVDProtocolExtension&mvd.ExtensionFloatCoords != 0 {
		readCoord = buf.GetCoord32
	}

	cmd.bits = bits

	bits &= ^uint16(511)

	if bits&protocol.UMoreBits != 0 {
		if cmd.MoreBits, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		bits |= uint16(cmd.MoreBits)
	}

	var moreBits uint16
	if bits&fte.UEvenMore != 0 && cmd.FTEProtocolExtension > 0 {
		if cmd.EvenMoreBits, err = buf.ReadByte(); err != nil {
			return nil, err
		}
		moreBits = uint16(cmd.EvenMoreBits)

		if cmd.EvenMoreBits&fte.UYetMore != 0 {
			if cmd.YetMoreBits, err = buf.ReadByte(); err != nil {
				return nil, err
			}
			yetMoreBits := uint16(cmd.YetMoreBits)

			moreBits |= yetMoreBits << 8
		}
	}

	if bits&protocol.UModel != 0 {
		if cmd.ModelIndex, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UFrame != 0 {
		if cmd.Frame, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UColorMap != 0 {
		if cmd.ColorMap, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.USkin != 0 {
		if cmd.Skin, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UEffects != 0 {
		if cmd.Effects, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UOrigin1 != 0 {
		if cmd.Coord[0], err = readCoord(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UAngle1 != 0 {
		if cmd.Angle[0], err = readAngle(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UOrigin2 != 0 {
		if cmd.Coord[1], err = readCoord(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UAngle2 != 0 {
		if cmd.Angle[1], err = readAngle(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UOrigin3 != 0 {
		if cmd.Coord[2], err = readCoord(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.UAngle3 != 0 {
		if cmd.Angle[2], err = readAngle(); err != nil {
			return nil, err
		}
	}

	if moreBits&fte.UTrans != 0 && cmd.FTEProtocolExtension&fte.ExtensionTrans != 0 {
		if cmd.Trans, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if moreBits&fte.UColorMod != 0 && cmd.FTEProtocolExtension&fte.ExtensionColorMod != 0 {
		for i := 0; i < len(cmd.ColorMod); i++ {
			if cmd.ColorMod[i], err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}
	}

	return &cmd, nil
}
