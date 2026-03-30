package svc

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet"
)

type Options struct {
	QWZCompatibility bool
}

func Parse(ctx *context.Context, data []byte) (packet.Packet, error) {
	buf := buffer.New(buffer.WithData(data))

	header, _ := buf.PeekInt32()
	if header == -1 {
		return parseConnectionless(ctx, buf)
	}

	return parseGameData(ctx, buf)
}

func ParseGameDataWithOptions(
	ctx *context.Context,
	data []byte,
	opts Options,
) (*GameData, error) {
	buf := buffer.New(buffer.WithData(data))
	return parseGameDataWithOptions(ctx, buf, opts)
}
