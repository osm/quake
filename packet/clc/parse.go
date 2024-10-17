package clc

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet"
)

func Parse(ctx *context.Context, data []byte) (packet.Packet, error) {
	buf := buffer.New(buffer.WithData(data))

	header, _ := buf.PeekInt32()
	if header == -1 {
		return parseConnectionless(ctx, buf)
	}

	return parseGameData(ctx, buf)
}
