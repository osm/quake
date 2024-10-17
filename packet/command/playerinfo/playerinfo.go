package playerinfo

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/deltausercommand"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
	"github.com/osm/quake/protocol/mvd"
)

const fteExtensions uint32 = fte.ExtensionHullSize |
	fte.ExtensionTrans |
	fte.ExtensionScale |
	fte.ExtensionFatness

type Command struct {
	IsMVD bool

	Index   byte
	Default *CommandDefault
	MVD     *CommandMVD
}

type CommandDefault struct {
	CoordSize            uint8
	MVDProtocolExtension uint32
	FTEProtocolExtension uint32

	Bits             uint16
	ExtraBits        byte
	Coord            [3]float32
	Frame            byte
	Msec             byte
	DeltaUserCommand *deltausercommand.Command
	Velocity         [3]uint16
	ModelIndex       byte
	SkinNum          byte
	Effects          byte
	WeaponFrame      byte
}

type CommandMVD struct {
	CoordSize uint8

	Bits        uint16
	Frame       byte
	Coord       [3]float32
	Angle       [3]float32
	ModelIndex  byte
	SkinNum     byte
	Effects     byte
	WeaponFrame byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCPlayerInfo)
	buf.PutByte(cmd.Index)

	if cmd.IsMVD && cmd.MVD != nil {
		buf.PutBytes(cmd.MVD.Bytes())
	} else if cmd.Default != nil {
		buf.PutBytes(cmd.Default.Bytes())
	}

	return buf.Bytes()
}

func (cmd *CommandDefault) Bytes() []byte {
	buf := buffer.New()

	writeCoord := buf.PutCoord16
	if cmd.CoordSize == 4 ||
		cmd.MVDProtocolExtension&mvd.ExtensionFloatCoords != 0 ||
		cmd.FTEProtocolExtension&fteExtensions != 0 {
		writeCoord = buf.PutCoord32
	}

	buf.PutUint16(cmd.Bits)

	if cmd.FTEProtocolExtension&fteExtensions != 0 && cmd.Bits&fte.UFarMore != 0 {
		buf.PutByte(cmd.ExtraBits)
	}

	for i := 0; i < 3; i++ {
		writeCoord(cmd.Coord[i])
	}

	buf.PutByte(cmd.Frame)

	if cmd.Bits&protocol.PFMsec != 0 {
		buf.PutByte(cmd.Msec)
	}

	if cmd.Bits&protocol.PFCommand != 0 {
		buf.PutBytes(cmd.DeltaUserCommand.Bytes())
	}

	for i := 0; i < 3; i++ {
		if cmd.Bits&(protocol.PFVelocity1<<i) != 0 {
			buf.PutUint16(cmd.Velocity[i])
		}
	}

	if cmd.Bits&protocol.PFModel != 0 {
		buf.PutByte(cmd.ModelIndex)
	}

	if cmd.Bits&protocol.PFSkinNum != 0 {
		buf.PutByte(cmd.SkinNum)
	}

	if cmd.Bits&protocol.PFEffects != 0 {
		buf.PutByte(cmd.Effects)
	}

	if cmd.Bits&protocol.PFWeaponFrame != 0 {
		buf.PutByte(cmd.WeaponFrame)
	}

	return buf.Bytes()
}

func (cmd *CommandMVD) Bytes() []byte {
	buf := buffer.New()

	writeCoord := buf.PutCoord16
	if cmd.CoordSize == 4 {
		writeCoord = buf.PutCoord32
	}

	buf.PutUint16(cmd.Bits)
	buf.PutByte(cmd.Frame)

	for i := 0; i < 3; i++ {
		if cmd.Bits&(protocol.DFOrigin<<i) != 0 {
			writeCoord(cmd.Coord[i])
		}
	}

	for i := 0; i < 3; i++ {
		if cmd.Bits&(protocol.DFAngles<<i) != 0 {
			buf.PutAngle16(cmd.Angle[i])
		}
	}

	if cmd.Bits&protocol.DFModel != 0 {
		buf.PutByte(cmd.ModelIndex)
	}

	if cmd.Bits&protocol.DFSkinNum != 0 {
		buf.PutByte(cmd.SkinNum)
	}

	if cmd.Bits&protocol.DFEffects != 0 {
		buf.PutByte(cmd.Effects)
	}

	if cmd.Bits&protocol.DFWeaponFrame != 0 {
		buf.PutByte(cmd.WeaponFrame)
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.IsMVD = ctx.GetIsMVD()

	if cmd.Index, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.IsMVD {
		if cmd.MVD, err = parseCommandMVD(ctx, buf); err != nil {
			return nil, err
		}
	} else {
		if cmd.Default, err = parseCommandDefault(ctx, buf); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}

func parseCommandDefault(ctx *context.Context, buf *buffer.Buffer) (*CommandDefault, error) {
	var err error
	var cmd CommandDefault

	cmd.FTEProtocolExtension = ctx.GetFTEProtocolExtension()
	cmd.MVDProtocolExtension = ctx.GetMVDProtocolExtension()
	cmd.CoordSize = ctx.GetCoordSize()

	readCoord := buf.GetCoord16

	if cmd.CoordSize == 4 ||
		cmd.MVDProtocolExtension&mvd.ExtensionFloatCoords != 0 ||
		cmd.FTEProtocolExtension&fteExtensions != 0 {
		readCoord = buf.GetCoord32
	}

	if cmd.Bits, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	var bits uint32 = uint32(cmd.Bits)

	if cmd.FTEProtocolExtension&fteExtensions != 0 && bits&fte.UFarMore != 0 {
		if cmd.ExtraBits, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		bits = (uint32(cmd.ExtraBits) << 16) | uint32(cmd.Bits)
	}

	for i := 0; i < 3; i++ {
		if cmd.Coord[i], err = readCoord(); err != nil {
			return nil, err
		}
	}

	if cmd.Frame, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if bits&protocol.PFMsec != 0 {
		if cmd.Msec, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.PFCommand != 0 {
		if cmd.DeltaUserCommand, err = deltausercommand.Parse(ctx, buf); err != nil {
			return nil, err
		}
	}

	for i := 0; i < 3; i++ {
		if bits&(protocol.PFVelocity1<<i) != 0 {
			if cmd.Velocity[i], err = buf.GetUint16(); err != nil {
				return nil, err
			}
		}
	}

	if bits&protocol.PFModel != 0 {
		if cmd.ModelIndex, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.PFSkinNum != 0 {
		if cmd.SkinNum, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.PFEffects != 0 {
		if cmd.Effects, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if bits&protocol.PFWeaponFrame != 0 {
		if cmd.WeaponFrame, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}

func parseCommandMVD(ctx *context.Context, buf *buffer.Buffer) (*CommandMVD, error) {
	var err error
	var cmd CommandMVD

	cmd.CoordSize = ctx.GetCoordSize()
	readCoord := buf.GetCoord16
	if cmd.CoordSize == 4 {
		readCoord = buf.GetCoord32
	}

	if cmd.Bits, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Frame, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		if cmd.Bits&(protocol.DFOrigin<<i) != 0 {
			if cmd.Coord[i], err = readCoord(); err != nil {
				return nil, err
			}
		}
	}

	for i := 0; i < 3; i++ {
		if cmd.Bits&(protocol.DFAngles<<i) != 0 {
			if cmd.Angle[i], err = buf.GetAngle16(); err != nil {
				return nil, err
			}
		}
	}

	if cmd.Bits&protocol.DFModel != 0 {
		if cmd.ModelIndex, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.DFSkinNum != 0 {
		if cmd.SkinNum, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.DFEffects != 0 {
		if cmd.Effects, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.DFWeaponFrame != 0 {
		if cmd.WeaponFrame, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
