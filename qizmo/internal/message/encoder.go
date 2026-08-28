package message

import (
	"fmt"
	"math"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/state"
)

type Encoder struct {
	ft       *freq.Tables
	state    *state.Packet
	qwdInput bool
}

func NewLinkEncoder(ft *freq.Tables, st *state.Packet) *Encoder {
	return &Encoder{ft: ft, state: st}
}

func NewQWDEncoder(ft *freq.Tables, st *state.Packet) *Encoder {
	return &Encoder{ft: ft, state: st, qwdInput: true}
}

type PacketPlan struct {
	encoder    *Encoder
	seq        uint32
	mode       byte
	base       packetBase
	primary    primaryPlayerPlan
	operations []operationPlan
}

type EncodingOptions struct {
	PreservePacketEntityDeltas bool
	// DisableHistoryReference encodes against the default state instead of a
	// preceding compressed packet.
	DisableHistoryReference bool
}

func (e *Encoder) PlanPacket(packet []byte, previousSeq uint32) (*PacketPlan, bool) {
	return e.PlanPacketWithOptions(packet, previousSeq, EncodingOptions{})
}

func (e *Encoder) PlanPacketWithOptions(
	packet []byte,
	previousSeq uint32,
	options EncodingOptions,
) (*PacketPlan, bool) {
	parsed, ok := e.parsePacket(packet, previousSeq, options)
	if !ok {
		return nil, false
	}
	return e.planPacket(parsed), true
}

func (p *PacketPlan) Mode() byte {
	return p.mode
}

func (p *PacketPlan) EncodeBody(enc *rangeenc.Encoder) error {
	e := p.encoder
	if err := e.encodeSVCPlayerInfo(enc, p.primary); err != nil {
		return err
	}
	context := encodingContext{
		primary:             p.primary.player,
		base:                p.base,
		currentPlayers:      []state.PlayerRecord{p.primary.result},
		lastCoordinates:     p.primary.player.origin,
		lastPingPlayerIndex: noPlayerIndex,
		lastPLPlayerIndex:   noPlayerIndex,
	}
	for i, operation := range p.operations {
		if err := e.encodeOperation(enc, &context, operation); err != nil {
			return fmt.Errorf("encode trailing svc %d opcode 0x%02x: %w", i, operation.opcode, err)
		}
	}
	if err := enc.EncodeSymbol(e.ft.CumulativeRow(freq.SVCType), protocol.SVCBad); err != nil {
		return fmt.Errorf("encode svc terminator: %w", err)
	}
	return nil
}

func (p *PacketPlan) Commit() {
	e := p.encoder
	players := []state.PlayerRecord{p.primary.result}
	var packetEntities []state.EntityRecord
	hasPacketEntities := false
	for _, plan := range p.operations {
		for _, player := range plan.playerPlans {
			players = append(players, player.result)
		}
		if plan.opcode == protocol.SVCPacketEntities {
			packetEntities = plan.packetEntities.records
			hasPacketEntities = true
		}
		trackPlayerIdentity(e.state, plan.opcode, plan.data)
	}

	e.state.BeginPacket(p.seq)
	if p.base.referenceDistance != 0 {
		e.state.SetPacketBase(p.base.referenceSequence)
	}
	e.state.CurrentPlayers = append(e.state.CurrentPlayers, players...)
	if hasPacketEntities {
		e.state.CommitPacketEntities(packetEntities, true)
	}
	e.state.CommitPacket()
}

func packetMode(previousSeq, seq uint32) (byte, bool) {
	delta := (seq - previousSeq) & protocol.QWSequenceMask
	if delta == 0 || delta > math.MaxUint8 {
		return 0, false
	}
	return byte(delta), true
}
