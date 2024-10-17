package modellist

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	NumModels byte
	Models    []string
	Index     byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCModelList)
	buf.PutByte(cmd.NumModels)

	for i := 0; i < len(cmd.Models); i++ {
		buf.PutString(cmd.Models[i])
	}
	buf.PutByte(0x00)

	buf.PutByte(cmd.Index)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.NumModels, err = buf.ReadByte(); err != nil {
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

	if cmd.Index, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
