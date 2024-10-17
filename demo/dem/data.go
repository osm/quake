package dem

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/svc"
)

type Data struct {
	Size   uint32
	Angle  [3]float32
	Packet packet.Packet
}

func (d *Data) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint32(d.Size)

	for i := 0; i < 3; i++ {
		buf.PutFloat32(d.Angle[i])
	}

	buf.PutBytes(d.Packet.Bytes())

	return buf.Bytes()
}

func parseData(ctx *context.Context, buf *buffer.Buffer) (*Data, error) {
	var err error
	var data Data

	if data.Size, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		if data.Angle[i], err = buf.GetFloat32(); err != nil {
			return nil, err
		}
	}

	bytes, err := buf.GetBytes(int(data.Size))
	if err != nil {
		return nil, err
	}

	if data.Packet, err = svc.Parse(ctx, bytes); err != nil {
		return nil, err
	}

	return &data, nil
}
