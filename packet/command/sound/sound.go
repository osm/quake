package sound

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	CoordSize uint8
	IsNQ      bool

	Bits        byte
	SoundIndex  byte
	Channel     uint16
	Volume      byte
	Attenuation byte
	SoundNum    byte
	Coord       [3]float32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCSound)

	if cmd.IsNQ {
		buf.PutByte(cmd.Bits)

		if cmd.Bits&protocol.NQSoundVolume != 0 {
			buf.PutByte(cmd.Volume)
		}

		if cmd.Bits&protocol.NQSoundAttenuation != 0 {
			buf.PutByte(cmd.Attenuation)
		}

		buf.PutUint16(cmd.Channel)
		buf.PutByte(cmd.SoundIndex)

		for i := 0; i < 3; i++ {
			buf.PutCoord16(cmd.Coord[i])
		}
	} else {
		writeCoord := buf.PutCoord16
		if cmd.CoordSize == 4 {
			writeCoord = buf.PutCoord32
		}

		buf.PutUint16(cmd.Channel)

		if cmd.Channel&protocol.SoundVolume != 0 {
			buf.PutByte(cmd.Volume)
		}

		if cmd.Channel&protocol.SoundAttenuation != 0 {
			buf.PutByte(cmd.Attenuation)
		}

		buf.PutByte(cmd.SoundNum)

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

	if cmd.IsNQ {
		if cmd.Bits, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		if cmd.Bits&protocol.NQSoundVolume != 0 {
			if cmd.Volume, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}

		if cmd.Bits&protocol.NQSoundAttenuation != 0 {
			if cmd.Attenuation, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}

		if cmd.Channel, err = buf.GetUint16(); err != nil {
			return nil, err
		}

		if cmd.SoundIndex, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		for i := 0; i < 3; i++ {
			if cmd.Coord[i], err = buf.GetCoord16(); err != nil {
				return nil, err
			}
		}

	} else {
		cmd.CoordSize = ctx.GetCoordSize()
		readCoord := buf.GetCoord16
		if cmd.CoordSize == 4 {
			readCoord = buf.GetCoord32
		}

		if cmd.Channel, err = buf.GetUint16(); err != nil {
			return nil, err
		}

		if cmd.Channel&protocol.SoundVolume != 0 {
			if cmd.Volume, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}

		if cmd.Channel&protocol.SoundAttenuation != 0 {
			if cmd.Attenuation, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}

		if cmd.SoundNum, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		for i := 0; i < 3; i++ {
			if cmd.Coord[i], err = readCoord(); err != nil {
				return nil, err
			}
		}
	}

	return &cmd, nil
}
