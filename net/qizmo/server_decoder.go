package qizmo

import (
	"encoding/binary"
	"fmt"

	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	qizmocodec "github.com/osm/quake/qizmo"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/standard"
	"github.com/osm/quake/qizmo/state"
)

type ServerDecoder struct {
	frequencies *freq.Tables
	packet      *state.Packet
	tracker     *standard.Tracker
	endpoint    *EndpointState
	sequences   qizmoprotocol.LinkSequences
}

func NewServerDecoder(endpoint *EndpointState) (*ServerDecoder, error) {
	frequencies, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		return nil, fmt.Errorf("load Qizmo frequency tables: %w", err)
	}
	return newServerDecoder(frequencies, assets.Embedded(), endpoint), nil
}

func newServerDecoder(
	frequencies *freq.Tables,
	packetAssets assets.Assets,
	endpoint *EndpointState,
) *ServerDecoder {
	packet := state.NewPacketWithAssets(0, packetAssets)

	return &ServerDecoder{
		frequencies: frequencies,
		packet:      packet,
		tracker:     standard.NewTracker(packet),
		endpoint:    endpoint,
	}
}

func (d *ServerDecoder) Decode(packet []byte) ([]byte, error) {
	if !qizmoprotocol.IsCompressedLinkPacket(packet) {
		if header, ok := readRawServerHeader(packet); ok {
			sequence := header.sequence & sequenceMask
			if err := d.tracker.Observe(packet, sequence); err != nil {
				return nil, fmt.Errorf("track raw server packet: %w", err)
			}
			d.sequences.Observe(header.sequence, header.ack)
			d.endpoint.observePlayerNames(d.packet.PlayerNames())
		}
		return packet, nil
	}

	header, body, err := d.sequences.DecodeHeader(packet)
	if err != nil {
		return nil, err
	}
	rangeDecoder := rangedec.NewPadded(body)

	decoded, err := qizmocodec.NewDecoder(
		rangeDecoder,
		d.frequencies,
		d.packet,
	).DecodePacketWithOptions(header.Sequence, header.Ack, qizmocodec.DecodingOptions{
		PreservePacketEntityDeltas: true,
	})
	if err != nil {
		return nil, fmt.Errorf("decode Qizmo server packet: %w", err)
	}
	d.endpoint.observePlayerNames(d.packet.PlayerNames())
	if decoded == nil {
		return nil, nil
	}

	// The shared message codec canonicalizes away the sequence reliable bit;
	// a network link must forward it to the downstream QuakeWorld client.
	binary.LittleEndian.PutUint32(decoded, header.Sequence)
	return decoded, nil
}

func (d *ServerDecoder) ObserveClientPacket(packet []byte) {
	observeClientMovement(d.packet, packet)
}
