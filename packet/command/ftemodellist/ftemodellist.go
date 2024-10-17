package ftemodellist

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol/fte"
)

type Command struct {
	NumModels uint16
	Models    []string
	More      byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(fte.SVCModelListShort)
	buf.PutUint16(cmd.NumModels)

	for i := 0; i < len(cmd.Models); i++ {
		buf.PutString(cmd.Models[i])
	}
	buf.PutByte(0x00)

	buf.PutByte(cmd.More)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.NumModels, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	for {
		var model string
		if model, err = buf.GetString(); err != nil {
			return nil, err
		}

		if model == "" {
			break
		}

		cmd.Models = append(cmd.Models, model)
	}

	if cmd.More, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
