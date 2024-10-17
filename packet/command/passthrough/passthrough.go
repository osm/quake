package passthrough

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type Command struct {
	Data []byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutBytes(cmd.Data)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer, str string) (*Command, error) {
	var cmd Command

	if len(str) > 0 {
		cmd.Data = []byte(str)
	}

	remaining := buf.Len() - buf.Off()
	if remaining > 0 {
		data, err := buf.GetBytes(remaining)
		if err != nil {
			return nil, err
		}

		cmd.Data = append(cmd.Data, data...)
	}

	return &cmd, nil
}
