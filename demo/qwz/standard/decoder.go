package standard

import (
	"fmt"
	"strings"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/demo/qwz/state"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/deltapacketentities"
	"github.com/osm/quake/packet/command/modellist"
	"github.com/osm/quake/packet/command/packetentities"
	"github.com/osm/quake/packet/command/playerinfo"
	"github.com/osm/quake/packet/command/qizmovoice"
	"github.com/osm/quake/packet/command/serverdata"
	"github.com/osm/quake/packet/command/soundlist"
	"github.com/osm/quake/packet/command/spawnbaseline"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/svc"
)

type decoder struct {
	r  *reader
	st *state.Packet

	handlers map[Event][]HandlerFunc

	packetEnts map[uint16]state.EntityRecord
}

type Event int

const (
	QizmoVoice Event = iota
)

type HandlerFunc func(payload []byte)

type Decoder struct {
	state    *state.Packet
	handlers map[Event][]HandlerFunc
	ctx      *context.Context
}

// New returns a QuakeWorld svc parser backed by the shared packet/svc command
// parser. The standard package remains responsible for applying the QWZ-specific
// packet state derived from those parsed commands.
func New(st *state.Packet) *Decoder {
	return &Decoder{
		state:    st,
		handlers: make(map[Event][]HandlerFunc),
		ctx:      context.New(),
	}
}

func (d *Decoder) HandleFunc(event Event, fn HandlerFunc) {
	d.handlers[event] = append(d.handlers[event], fn)
}

func (d *Decoder) Decode(packet []byte, seq uint32) error {
	if len(packet) < 8 {
		return fmt.Errorf("raw packet too short: %d", len(packet))
	}

	if seq == 0xffffffff {
		return nil
	}

	pkg, err := svc.ParseGameDataWithOptions(
		d.ctx,
		packet,
		svc.Options{
			QWZCompatibility: true,
		},
	)
	if err != nil {
		return err
	}

	parser := &decoder{
		st:       d.state,
		handlers: d.handlers,
	}

	parser.st.BeginPacket(seq)

	for i, parsedCmd := range pkg.Commands {
		opcode := pkg.RawCmds[i][0]
		if opcode == 0x3e {
			return nil
		}
		if opcode >= 0x4a && opcode != 0x51 && opcode != 0x52 && opcode != 0x53 {
			break
		}
		if err := parser.applyCommand(parsedCmd, pkg.RawCmds[i], seq); err != nil {
			return err
		}
	}

	parser.finalizePacket(seq)
	return nil
}

func (d *decoder) applyCommand(cmdAny command.Command, raw []byte, seq uint32) error {
	switch cmd := cmdAny.(type) {
	case *serverdata.Command:
		d.st.PlayerIndex = cmd.PlayerNumber & 0x7f
	case *stufftext.Command:
		if !d.st.TablesBuilt &&
			(strings.HasPrefix(cmd.String, "cmd prespawn") ||
				strings.HasPrefix(cmd.String, "cmd spawn")) {
			d.st.RebuildRemaps()
		}
	case *spawnbaseline.Command:
		return d.applySpawnBaseline(raw)
	case *playerinfo.Command:
		return d.applyPlayerInfo(raw, seq)
	case *packetentities.Command:
		return d.applyPacketEntities(raw)
	case *deltapacketentities.Command:
		return d.applyDeltaPacketEntities(raw, cmd.Index)
	case *modellist.Command:
		if err := d.applyModelList(raw); err != nil {
			return err
		}
		d.st.AddModelChunk(raw)
	case *soundlist.Command:
		if err := d.applySoundList(raw); err != nil {
			return err
		}
		d.st.AddSoundChunk(raw)
	case *qizmovoice.Command:
		for _, handler := range d.handlers[QizmoVoice] {
			handler(append([]byte(nil), cmd.Data...))
		}
	}

	return nil
}

func (d *decoder) applySpawnBaseline(raw []byte) error {
	d.r = newReader(raw[1:])
	return d.parseSpawnBaseline()
}

func (d *decoder) applyPlayerInfo(raw []byte, seq uint32) error {
	d.r = newReader(raw[1:])
	return d.parsePlayerInfo(seq)
}

func (d *decoder) applyPacketEntities(raw []byte) error {
	d.r = newReader(raw[1:])
	return d.parsePacketEntities(nil)
}

func (d *decoder) applyDeltaPacketEntities(raw []byte, ref byte) error {
	base, _ := d.st.FindRawEntitiesByte(ref)
	d.r = newReader(raw[2:])
	return d.parsePacketEntities(base)
}

func (d *decoder) applyModelList(raw []byte) error {
	r := newReader(raw[1:])
	start, err := r.ReadByte()
	if err != nil {
		return err
	}
	if start == 0 {
		d.st.ResetEntityTracking()
	}

	modelIdx := int(start)
	for {
		s, err := r.ReadString()
		if err != nil {
			return err
		}
		if len(s) == 1 {
			return nil
		}

		modelIdx++
		name := string(s[:len(s)-1])
		d.st.ModelNames[modelIdx] = name
		if name == "progs/player.mdl" {
			d.st.PlayerFreqIndex = byte(modelIdx)
		}
	}
}

func (d *decoder) applySoundList(raw []byte) error {
	r := newReader(raw[1:])
	start, err := r.ReadByte()
	if err != nil {
		return err
	}

	soundIdx := int(start)
	for {
		s, err := r.ReadString()
		if err != nil {
			return err
		}
		if len(s) == 1 {
			return nil
		}

		soundIdx++
		d.st.SoundNames[soundIdx] = string(s[:len(s)-1])
	}
}

func (d *decoder) finalizePacket(seq uint32) {
	histSeq := seq
	if len(d.st.CurrentPlayers) != 0 && d.st.CmdSeqNo != 0 {
		histSeq = d.st.CmdSeqNo + 1
	}

	if d.packetEnts != nil {
		for entNum, rec := range d.packetEnts {
			d.st.EntityLast[entNum] = rec
			d.st.EntityRaw[entNum] = rec
			d.st.EntityLastRaw[entNum] = true
		}

		d.st.CommitRawEntitiesByte(byte(seq), d.packetEnts)
		d.st.CommitEntities(histSeq, d.packetEnts)
	}

	d.st.CommitPacketAs(histSeq)
}
