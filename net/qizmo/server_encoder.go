package qizmo

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	qizmocodec "github.com/osm/quake/qizmo"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/standard"
	"github.com/osm/quake/qizmo/state"
)

type ServerEncoder struct {
	packet         *state.Packet
	tracker        *standard.Tracker
	messageEncoder *qizmocodec.Encoder
	endpoint       *EndpointState

	sequences sequenceTracker
}

func NewServerEncoder(endpoint *EndpointState) (*ServerEncoder, error) {
	frequencies, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		return nil, fmt.Errorf("load Qizmo frequency tables: %w", err)
	}
	return newServerEncoder(frequencies, assets.Embedded(), endpoint), nil
}

func newServerEncoder(
	frequencies *freq.Tables,
	packetAssets assets.Assets,
	endpoint *EndpointState,
) *ServerEncoder {
	packet := state.NewPacketWithAssets(0, packetAssets)
	return &ServerEncoder{
		packet:         packet,
		tracker:        standard.NewTracker(packet),
		messageEncoder: qizmocodec.NewLinkEncoder(frequencies, packet),
		endpoint:       endpoint,
	}
}

func (e *ServerEncoder) Encode(packet []byte) ([]byte, error) {
	rawHeader, ok := readRawServerHeader(packet)
	if !ok {
		return packet, nil
	}

	sequence := rawHeader.sequence & sequenceMask
	previousSequence := e.sequences.last
	disableHistoryReference := e.endpoint.needsS2CResync()

	// The compressed message does not encode acknowledgements or reliable bits;
	// the surrounding link header preserves both. Normalize only the temporary
	// packet presented to the shared message encoder.
	messagePacket := append([]byte(nil), packet...)
	binary.LittleEndian.PutUint32(messagePacket, sequence)
	binary.LittleEndian.PutUint32(messagePacket[protocol.QWPacketAckOffset:], sequence)
	plan, supported := e.messageEncoder.PlanPacketWithOptions(
		messagePacket,
		previousSequence,
		qizmocodec.EncodingOptions{
			PreservePacketEntityDeltas: true,
			DisableHistoryReference:    disableHistoryReference,
		},
	)

	var compressed []byte
	if supported {
		rangeEncoder := rangeenc.New()
		if err := plan.EncodeBody(rangeEncoder); err != nil {
			return nil, fmt.Errorf("encode compressed server packet: %w", err)
		}
		header := qizmoprotocol.EncodeHeader(qizmoprotocol.LinkHeader{
			Sequence: rawHeader.sequence,
			Ack:      rawHeader.ack,
		})
		compressed = append(header[:], rangeEncoder.Finish()...)
	}

	if len(compressed) != 0 && len(compressed) < len(packet) {
		e.sequences.observe(sequence)
		plan.Commit()
		e.endpoint.observePlayerNames(e.packet.PlayerNames())
		e.endpoint.clearS2CResyncRequest()
		return compressed, nil
	}
	if err := e.ObserveRawPacket(packet); err != nil {
		return nil, err
	}
	e.endpoint.clearS2CResyncRequest()
	return packet, nil
}

func (e *ServerEncoder) ObserveRawPacket(packet []byte) error {
	header, ok := readRawServerHeader(packet)
	if !ok {
		return nil
	}
	sequence := header.sequence & sequenceMask
	if err := e.tracker.Observe(packet, sequence); err != nil {
		return fmt.Errorf("track raw server packet: %w", err)
	}
	e.sequences.observe(sequence)
	e.endpoint.observePlayerNames(e.packet.PlayerNames())
	return nil
}

func (e *ServerEncoder) ObserveClientPacket(packet []byte) {
	observeClientMovement(e.packet, packet)
}
