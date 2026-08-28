package standard

import (
	"fmt"
	"strings"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/deltapacketentities"
	"github.com/osm/quake/packet/command/modellist"
	"github.com/osm/quake/packet/command/packetentities"
	"github.com/osm/quake/packet/command/playerinfo"
	"github.com/osm/quake/packet/command/qizmovoice"
	"github.com/osm/quake/packet/command/serverdata"
	"github.com/osm/quake/packet/command/setinfo"
	"github.com/osm/quake/packet/command/soundlist"
	"github.com/osm/quake/packet/command/spawnbaseline"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/command/updatename"
	"github.com/osm/quake/packet/command/updateuserinfo"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/state"
)

type packetParser struct {
	reader *reader
	state  *state.Packet

	voiceHandlers []func([]byte)

	packetEntities map[uint16]state.EntityRecord
}

type Tracker struct {
	state         *state.Packet
	voiceHandlers []func([]byte)
	ctx           *context.Context
}

func NewTracker(packetState *state.Packet) *Tracker {
	return &Tracker{
		state: packetState,
		ctx:   context.New(),
	}
}

func (t *Tracker) HandleVoice(fn func([]byte)) {
	t.voiceHandlers = append(t.voiceHandlers, fn)
}

func (t *Tracker) Observe(packet []byte, sequence uint32) error {
	if len(packet) < protocol.QWServerPacketHeaderSize {
		return fmt.Errorf("raw packet too short: %d", len(packet))
	}

	if sequence == protocol.QWConnectionlessSequence {
		return nil
	}

	parsedPacket, err := svc.ParseGameDataWithOptions(
		t.ctx,
		packet,
		svc.Options{
			QizmoCompatibility: true,
		},
	)
	if err != nil {
		return err
	}

	parser := &packetParser{
		state:         t.state,
		voiceHandlers: t.voiceHandlers,
	}

	parser.state.BeginPacket(sequence)

	for i, parsedCommand := range parsedPacket.Commands {
		if err := parser.applyCommand(parsedCommand, parsedPacket.RawCmds[i], sequence); err != nil {
			return err
		}
	}

	parser.finalizePacket(sequence)
	return nil
}

func (p *packetParser) applyCommand(parsed command.Command, raw []byte, sequence uint32) error {
	switch cmd := parsed.(type) {
	case *serverdata.Command:
		p.state.PlayerIndex = cmd.PlayerNumber & protocol.QWServerDataPlayerMask
		p.state.ResetPlayerNames()
		p.state.InvalidateHistory(sequence)
	case *stufftext.Command:
		if !p.state.RemapsReady() &&
			(strings.HasPrefix(cmd.String, "cmd prespawn") ||
				strings.HasPrefix(cmd.String, "cmd spawn")) {
			p.state.RebuildRemaps()
		}
	case *updateuserinfo.Command:
		p.state.SetPlayerUserInfo(cmd.PlayerIndex, cmd.UserInfo)
	case *setinfo.Command:
		if cmd.Key == "name" {
			p.state.SetPlayerName(cmd.PlayerIndex, cmd.Value)
		}
	case *updatename.Command:
		p.state.SetPlayerName(cmd.PlayerIndex, cmd.Name)
	case *spawnbaseline.Command:
		return p.applySpawnBaseline(raw)
	case *playerinfo.Command:
		return p.applyPlayerInfo(raw)
	case *packetentities.Command:
		return p.applyPacketEntities(raw)
	case *deltapacketentities.Command:
		return p.applyDeltaPacketEntities(raw, cmd.Index, sequence)
	case *modellist.Command:
		p.applyModelList(cmd)
		p.state.AddModelChunk(raw)
	case *soundlist.Command:
		p.state.AddSoundChunk(raw)
	case *qizmovoice.Command:
		for _, handler := range p.voiceHandlers {
			handler(append([]byte(nil), cmd.Data...))
		}
	}

	return nil
}

func (p *packetParser) applySpawnBaseline(raw []byte) error {
	p.reader = newReader(raw[1:])
	return p.parseSpawnBaseline()
}

func (p *packetParser) applyPlayerInfo(raw []byte) error {
	p.reader = newReader(raw[1:])
	return p.parsePlayerInfo()
}

func (p *packetParser) applyPacketEntities(raw []byte) error {
	p.reader = newReader(raw[1:])
	return p.parsePacketEntities(nil)
}

func (p *packetParser) applyDeltaPacketEntities(raw []byte, reference byte, sequence uint32) error {
	base, ok := p.state.RawEntitySnapshot(reference)
	if !ok {
		p.state.InvalidateHistory(sequence)
	}
	p.reader = newReader(raw[2:])
	return p.parsePacketEntities(base)
}

func (p *packetParser) applyModelList(cmd *modellist.Command) {
	if cmd.NumModels == 0 {
		p.state.ResetEntityTracking()
	}

	for i, name := range cmd.Models {
		if name == "progs/player.mdl" {
			p.state.PlayerModelIndex = byte(int(cmd.NumModels) + i + 1)
		}
	}
}

func (p *packetParser) finalizePacket(sequence uint32) {
	historySequence := sequence
	if len(p.state.CurrentPlayers) != 0 && p.state.CommandSequence != 0 {
		historySequence = p.state.CommandSequence + 1
	}

	if p.packetEntities != nil {
		p.state.CommitRawEntitySnapshot(byte(sequence), p.packetEntities)
		p.state.CommitEntitySnapshot(historySequence, p.packetEntities)
	}

	p.state.CommitPlayerSnapshot(historySequence)
}
