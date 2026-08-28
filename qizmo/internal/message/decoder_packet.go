package message

import (
	"errors"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/state"
)

func (d *Decoder) DecodePacket(
	seq uint32,
	ack uint32,
) ([]byte, error) {
	return d.DecodePacketWithOptions(seq, ack, DecodingOptions{})
}

func (d *Decoder) DecodePacketWithOptions(
	seq uint32,
	ack uint32,
	options DecodingOptions,
) ([]byte, error) {
	d.endedOnDroppedPacket = false
	streamStart := *d.stream
	rd := *d.stream
	packetDecoder := &packetDecoder{
		rd:                  &rd,
		ft:                  d.ft,
		state:               d.state,
		lastPlayerIndex:     noPlayerIndex,
		lastPingPlayerIndex: noPlayerIndex,
		lastPLPlayerIndex:   noPlayerIndex,
	}

	var out []byte

	d.state.BeginPacket(seq)

	out = appendUint32LE(out, seq&protocol.QWSequenceMask)
	out = appendUint32LE(out, ack)

	out = append(out, protocol.SVCPlayerInfo)
	body, err := packetDecoder.decodeSVCPlayerInfo()
	if err != nil {
		if errors.Is(err, errDroppedPacket) {
			*d.stream = rd
			return nil, nil
		}
		return nil, err
	}
	out = append(out, body...)
	packetDecoder.refreshPacketContext()
	hasTrailingSVC := false

	for {
		opcode, err := packetDecoder.rd.DecodeFreqByte(
			packetDecoder.ft,
			freq.SVCType,
		)
		if err != nil {
			return nil, err
		}

		if opcode == protocol.SVCBad {
			break
		}

		hasTrailingSVC = true
		out = append(out, opcode)
		out, err = packetDecoder.decodeOperation(out, opcode, options)
		if err != nil {
			if errors.Is(err, errDroppedPacket) {
				*d.stream = rd
				return nil, nil
			}
			return nil, err
		}
	}

	if options.TreatPrimaryOnlyPacketAsDropped && !hasTrailingSVC {
		d.endedOnDroppedPacket = true
		d.state.CommitPacket()
		*d.stream = streamStart
		return nil, nil
	}

	d.state.CommitPacket()
	*d.stream = rd

	return out, nil
}

func (d *packetDecoder) refreshPacketContext() {
	if len(d.state.CurrentPlayers) != 0 {
		lastRec := d.state.CurrentPlayers[len(d.state.CurrentPlayers)-1]
		d.lastCoordinates = state.PlayerOrigin(lastRec)

		firstRec := d.state.CurrentPlayers[0]
		d.primaryCoordinates = state.PlayerOrigin(firstRec)
	}

	if baseSequence, ok := d.state.PacketBase(); ok {
		if players, ok := d.state.PlayerSnapshot(baseSequence); ok {
			d.basePlayers = players
		} else {
			d.basePlayers = nil
		}
		if entities, ok := d.state.EntitySnapshot(baseSequence); ok {
			d.baseEntities = entities
		} else {
			d.baseEntities = nil
		}
		d.packetScale =
			int(d.state.CommandScale(baseSequence)) *
				int(d.state.Sequence()-baseSequence)
	} else {
		d.basePlayers = nil
		d.baseEntities = nil
		d.packetScale = 0
	}
}
