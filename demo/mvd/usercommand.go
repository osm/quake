package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type UserCommand struct {
	size uint32

	Data        []byte
	PlayerIndex byte
	DropIndex   byte
	Msec        byte
	Angle       [3]float32
	Forward     uint16
	Side        uint16
	Up          uint16
	Buttons     byte
	Impulse     byte
}

func (cmd *UserCommand) Bytes() []byte {
	if cmd.size != 23 {
		return cmd.Data
	}

	buf := buffer.New()

	buf.PutByte(cmd.PlayerIndex)
	buf.PutByte(cmd.DropIndex)
	buf.PutByte(cmd.Msec)

	for _, a := range cmd.Angle {
		buf.PutFloat32(a)
	}

	buf.PutUint16(cmd.Forward)
	buf.PutUint16(cmd.Side)
	buf.PutUint16(cmd.Up)
	buf.PutByte(cmd.Buttons)
	buf.PutByte(cmd.Impulse)

	return buf.Bytes()
}

func parseUserCommand(ctx *context.Context, buf *buffer.Buffer, size uint32) (*UserCommand, error) {
	var err error
	var cmd UserCommand

	cmd.size = size
	if cmd.size != 23 {
		if cmd.Data, err = buf.GetBytes(int(cmd.size)); err != nil {
			return nil, err
		}

		goto end
	}

	if cmd.PlayerIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.DropIndex, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Msec, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		if cmd.Angle[i], err = buf.GetFloat32(); err != nil {
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

end:
	return &cmd, nil
}
