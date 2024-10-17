package qwd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/svc"
)

type Read struct {
	Size   uint32
	Packet packet.Packet
}

func (cmd *Read) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint32(cmd.Size)
	buf.PutBytes(cmd.Packet.Bytes())

	return buf.Bytes()
}

func parseRead(ctx *context.Context, buf *buffer.Buffer) (*Read, error) {
	var err error
	var cmd Read

	if cmd.Size, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	bytes, err := buf.GetBytes(int(cmd.Size))
	if err != nil {
		return nil, err
	}

	if cmd.Packet, err = svc.Parse(ctx, bytes); err != nil {
		return nil, err
	}

	return &cmd, nil
}
