package baseline

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type Command struct {
	AngleSize uint8
	CoordSize uint8

	ModelIndex byte
	Frame      byte
	ColorMap   byte
	SkinNum    byte
	Coord      [3]float32
	Angle      [3]float32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	writeAngle := buf.PutAngle8
	if cmd.AngleSize == 2 {
		writeAngle = buf.PutAngle16
	}

	writeCoord := buf.PutCoord16
	if cmd.CoordSize == 4 {
		writeCoord = buf.PutCoord32
	}

	buf.PutByte(cmd.ModelIndex)
	buf.PutByte(cmd.Frame)
	buf.PutByte(cmd.ColorMap)
	buf.PutByte(cmd.SkinNum)

	for i := 0; i < 3; i++ {
		writeCoord(cmd.Coord[i])
		writeAngle(cmd.Angle[i])
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.AngleSize = ctx.GetAngleSize()
	readAngle := buf.GetAngle8
	if cmd.AngleSize == 2 {
		readAngle = buf.GetAngle16
	}

	cmd.CoordSize = ctx.GetCoordSize()
	readCoord := buf.GetCoord16
	if cmd.CoordSize == 4 {
		readCoord = buf.GetCoord32
	}

	if cmd.ModelIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Frame, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.ColorMap, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.SkinNum, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		if cmd.Coord[i], err = readCoord(); err != nil {
			return nil, err
		}

		if cmd.Angle[i], err = readAngle(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}