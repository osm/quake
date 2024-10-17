package qwd

import (
	"errors"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

var ErrUnknownType = errors.New("unknown type")

type Demo struct {
	Data []*Data
}

func (dem *Demo) Bytes() []byte {
	buf := buffer.New()

	for _, d := range dem.Data {
		buf.PutBytes(d.Bytes())
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, data []byte) (*Demo, error) {
	var err error
	var cmd Demo

	buf := buffer.New(buffer.WithData(data))
	ctx.SetIsQWD(true)

	for buf.Off() < buf.Len() {
		var data Data

		if data.Timestamp, err = buf.GetFloat32(); err != nil {
			return nil, err
		}

		if data.Command, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		switch data.Command {
		case protocol.DemoCmd:
			if data.Cmd, err = parseCmd(ctx, buf); err != nil {
				return nil, err
			}
		case protocol.DemoRead:
			if data.Read, err = parseRead(ctx, buf); err != nil {
				return nil, err
			}
		case protocol.DemoSet:
			if data.Set, err = parseSet(ctx, buf); err != nil {
				return nil, err
			}
		default:
			return nil, ErrUnknownType
		}

		cmd.Data = append(cmd.Data, &data)
	}

	return &cmd, nil
}
