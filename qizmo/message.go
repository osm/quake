package qizmo

import (
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/message"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/state"
)

type Encoder = message.Encoder

type Decoder = message.Decoder

type DecodingOptions = message.DecodingOptions

type EncodingOptions = message.EncodingOptions

type PacketPlan = message.PacketPlan

func NewLinkEncoder(frequencies *freq.Tables, packetState *state.Packet) *Encoder {
	return message.NewLinkEncoder(frequencies, packetState)
}

func NewQWDEncoder(frequencies *freq.Tables, packetState *state.Packet) *Encoder {
	return message.NewQWDEncoder(frequencies, packetState)
}

func NewDecoder(
	stream *rangedec.Decoder,
	frequencies *freq.Tables,
	packetState *state.Packet,
) *Decoder {
	return message.NewDecoder(stream, frequencies, packetState)
}
