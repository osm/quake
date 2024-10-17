package tempentity

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	IsNQ      bool
	CoordSize uint8

	Type byte

	Coord       [3]float32
	EndCoord    [3]float32
	Entity      uint16
	Count       byte
	ColorStart  byte
	ColorLength byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	writeCoord := buf.PutCoord16
	if cmd.CoordSize == 4 {
		writeCoord = buf.PutCoord32
	}

	buf.PutByte(protocol.SVCTempEntity)
	buf.PutByte(cmd.Type)

	switch cmd.Type {
	case protocol.TELightning1:
		fallthrough
	case protocol.TELightning2:
		fallthrough
	case protocol.TELightning3:
		buf.PutUint16(cmd.Entity)

		for i := 0; i < 3; i++ {
			writeCoord(cmd.Coord[i])
		}

		for i := 0; i < 3; i++ {
			writeCoord(cmd.EndCoord[i])
		}
	case protocol.TEGunshot:
		fallthrough
	case protocol.TEBlood:
		if !cmd.IsNQ {
			buf.PutByte(cmd.Count)
		}

		for i := 0; i < 3; i++ {
			writeCoord(cmd.Coord[i])
		}

		if cmd.IsNQ && cmd.Type != protocol.TEGunshot {
			buf.PutByte(cmd.ColorStart)
			buf.PutByte(cmd.ColorLength)
		}
	case protocol.TELightningBlood:
		if cmd.IsNQ {
			buf.PutUint16(cmd.Entity)
		}

		for i := 0; i < 3; i++ {
			writeCoord(cmd.Coord[i])
		}

		if cmd.IsNQ {
			for i := 0; i < 3; i++ {
				writeCoord(cmd.EndCoord[i])
			}
		}
	default:
		for i := 0; i < 3; i++ {
			writeCoord(cmd.Coord[i])
		}
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.IsNQ = ctx.GetIsNQ()

	cmd.CoordSize = ctx.GetCoordSize()
	readCoord := buf.GetCoord16
	if cmd.CoordSize == 4 {
		readCoord = buf.GetCoord32
	}

	if cmd.Type, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	switch cmd.Type {
	case protocol.TELightning1:
		fallthrough
	case protocol.TELightning2:
		fallthrough
	case protocol.TELightning3:
		if cmd.Entity, err = buf.GetUint16(); err != nil {
			return nil, err
		}

		for i := 0; i < 3; i++ {
			if cmd.Coord[i], err = readCoord(); err != nil {
				return nil, err
			}
		}
		for i := 0; i < 3; i++ {
			if cmd.EndCoord[i], err = readCoord(); err != nil {
				return nil, err
			}
		}
	case protocol.TEGunshot:
		fallthrough
	case protocol.TEBlood:
		if !cmd.IsNQ {
			if cmd.Count, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}

		for i := 0; i < 3; i++ {
			if cmd.Coord[i], err = readCoord(); err != nil {
				return nil, err
			}
		}

		if cmd.IsNQ && cmd.Type != protocol.TEGunshot {
			if cmd.ColorStart, err = buf.ReadByte(); err != nil {
				return nil, err
			}

			if cmd.ColorLength, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}
	case protocol.TELightningBlood:
		if cmd.IsNQ {
			if cmd.Entity, err = buf.GetUint16(); err != nil {
				return nil, err
			}
		}

		for i := 0; i < 3; i++ {
			if cmd.Coord[i], err = readCoord(); err != nil {
				return nil, err
			}
		}

		if cmd.IsNQ {
			for i := 0; i < 3; i++ {
				if cmd.EndCoord[i], err = readCoord(); err != nil {
					return nil, err
				}
			}
		}
	default:
		for i := 0; i < 3; i++ {
			if cmd.Coord[i], err = readCoord(); err != nil {
				return nil, err
			}
		}
	}

	return &cmd, nil
}
