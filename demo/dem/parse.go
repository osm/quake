package dem

import (
	"errors"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

var ErrUnknownType = errors.New("unknown type")

type Demo struct {
	CDTrack []byte
	Data    []*Data
}

func (dem *Demo) Bytes() []byte {
	buf := buffer.New()

	buf.PutBytes(dem.CDTrack)

	for _, d := range dem.Data {
		buf.PutBytes(d.Bytes())
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, data []byte) (*Demo, error) {
	var demo Demo

	buf := buffer.New(buffer.WithData(data))
	ctx.SetIsDem(true)

	for buf.Off() < buf.Len() {
		b, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}

		demo.CDTrack = append(demo.CDTrack, b)
		if b == '\n' {
			break
		}
	}

	for buf.Off() < buf.Len() {
		d, err := parseData(ctx, buf)
		if err != nil {
			return nil, err
		}

		demo.Data = append(demo.Data, d)
	}

	return &demo, nil
}
