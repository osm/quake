package message

import (
	"errors"

	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/state"
)

var errDroppedPacket = errors.New("dropped compressed packet")

type packetDecoder struct {
	rd    *rangedec.Decoder
	ft    *freq.Tables
	state *state.Packet

	lastEntity          uint16
	lastCoordinates     [3]uint16
	lastPlayerIndex     byte
	lastPingPlayerIndex byte
	lastPLPlayerIndex   byte

	primaryCoordinates [3]uint16
	basePlayers        []state.PlayerRecord
	baseEntities       []state.EntityRecord
	packetScale        int
}

type Decoder struct {
	stream *rangedec.Decoder
	ft     *freq.Tables
	state  *state.Packet

	endedOnDroppedPacket bool
}

type DecodingOptions struct {
	TreatPrimaryOnlyPacketAsDropped bool
	PreservePacketEntityDeltas      bool
}

func NewDecoder(
	stream *rangedec.Decoder,
	ft *freq.Tables,
	st *state.Packet,
) *Decoder {
	return &Decoder{
		stream: stream,
		ft:     ft,
		state:  st,
	}
}

func (d *Decoder) EndedOnDroppedPacket() bool {
	return d.endedOnDroppedPacket
}
