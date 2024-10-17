package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type Cmd struct {
	Msec      byte
	UserAngle [3]float32
	Forward   uint16
	Side      uint16
	Up        uint16
	Buttons   byte
	Impulse   byte
	Padding   [3]byte
	Angle     [3]float32
}

func (cmd *Cmd) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(cmd.Msec)

	for i := 0; i < 3; i++ {
		buf.PutFloat32(cmd.UserAngle[i])
	}

	buf.PutUint16(cmd.Forward)
	buf.PutUint16(cmd.Side)
	buf.PutUint16(cmd.Up)
	buf.PutByte(cmd.Buttons)
	buf.PutByte(cmd.Impulse)

	for i := 0; i < 3; i++ {
		buf.PutByte(cmd.Padding[i])
	}

	for i := 0; i < 3; i++ {
		buf.PutFloat32(cmd.Angle[i])
	}

	return buf.Bytes()
}

func parseCmd(ctx *context.Context, buf *buffer.Buffer) (*Cmd, error) {
	var err error
	var cmd Cmd

	if cmd.Msec, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		if cmd.UserAngle[i], err = buf.GetFloat32(); err != nil {
			return nil, err
		}
	}

	if cmd.Forward, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Side, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Up, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Buttons, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Impulse, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		if cmd.Padding[i], err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	for i := 0; i < 3; i++ {
		if cmd.Angle[i], err = buf.GetFloat32(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
