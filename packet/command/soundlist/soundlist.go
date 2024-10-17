package soundlist

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	NumSounds byte
	Sounds    []string
	Index     byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCSoundList)
	buf.PutByte(cmd.NumSounds)

	for i := 0; i < len(cmd.Sounds); i++ {
		buf.PutString(cmd.Sounds[i])
	}
	buf.PutByte(0x00)

	buf.PutByte(cmd.Index)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.NumSounds, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	for {
		var sound string
		if sound, err = buf.GetString(); err != nil {
			return nil, err
		}

		if sound == "" {
			break
		}

		cmd.Sounds = append(cmd.Sounds, sound)
	}

	if cmd.Index, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
