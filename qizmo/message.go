package qizmo

import (
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/message"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/state"
)

type Decoder = message.Decoder

type DecodingOptions = message.DecodingOptions

func NewDecoder(
	stream *rangedec.Decoder,
	frequencies *freq.Tables,
	packetState *state.Packet,
) *Decoder {
	return message.NewDecoder(stream, frequencies, packetState)
}
