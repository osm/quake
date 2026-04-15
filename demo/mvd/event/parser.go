package event

import (
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/demo/mvd"
	"github.com/osm/quake/packet/command/playerinfo"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/command/updatefrags"
	"github.com/osm/quake/packet/command/updatestat"
	"github.com/osm/quake/packet/command/updatestatlong"
	"github.com/osm/quake/packet/command/updateuserinfo"
	"github.com/osm/quake/packet/svc"
)

type parser struct {
	elapsed       float64
	sawMatchStart bool
	matchRunning  bool

	playerNames  map[byte]string
	playerTeams  map[byte]string
	slotsByName  map[string]byte
	positions    map[byte]Vec3
	viewAngles   map[byte]Vec3
	lastSample   map[byte]Vec3
	packetSlot   byte
	maxClients   int
	mapName      string
	itemFlags    map[byte]int32
	activeWeapon map[byte]int32
	weaponFrames map[byte]byte
	healthBySlot map[byte]int
	armorBySlot  map[byte]int
	fragsBySlot  map[byte]int

	events          []Event
	matchStartIndex int
}

func Parse(data []byte) (*Result, error) {
	state := newParser()
	if err := state.parse(data); err != nil {
		return nil, err
	}
	return state.result(), nil
}

func newParser() *parser {
	return &parser{
		playerNames:     make(map[byte]string),
		playerTeams:     make(map[byte]string),
		slotsByName:     make(map[string]byte),
		positions:       make(map[byte]Vec3),
		viewAngles:      make(map[byte]Vec3),
		lastSample:      make(map[byte]Vec3),
		itemFlags:       make(map[byte]int32),
		activeWeapon:    make(map[byte]int32),
		weaponFrames:    make(map[byte]byte),
		healthBySlot:    make(map[byte]int),
		armorBySlot:     make(map[byte]int),
		fragsBySlot:     make(map[byte]int),
		matchStartIndex: -1,
	}
}

func (p *parser) parse(data []byte) error {
	demoFile, err := mvd.Parse(context.New(), data)
	if err != nil {
		return err
	}

	for _, packetData := range demoFile.Data {
		if p.matchRunning {
			p.elapsed += float64(packetData.Timestamp) * 0.001
		}

		if packetData.Read == nil {
			continue
		}

		gameData, ok := packetData.Read.Packet.(*svc.GameData)
		if !ok {
			continue
		}
		p.packetSlot = byte(packetData.Target)

		for _, command := range gameData.Commands {
			switch c := command.(type) {
			case *updateuserinfo.Command:
				p.handleUpdateUserInfo(c)
			case *stufftext.Command:
				p.handleStuffText(c)
			case *playerinfo.Command:
				p.handlePlayerInfo(c)
			case *updatefrags.Command:
				p.handleUpdateFrags(c)
			case *updatestat.Command:
				p.handleUpdateStat(c)
			case *updatestatlong.Command:
				p.handleUpdateStatLong(c)
			case *print.Command:
				p.handlePrint(c)
			}
		}
	}

	return nil
}

func (p *parser) result() *Result {
	return &Result{
		Events:     p.timeline(),
		maxClients: p.maxClients,
		mapName:    p.mapName,
	}
}

func (p *parser) timeline() []Event {
	if p.sawMatchStart && p.matchStartIndex >= 0 &&
		p.matchStartIndex < len(p.events) {
		matchEvents := p.events[p.matchStartIndex:]
		if len(matchEvents) > 0 {
			return matchEvents
		}
	}
	return p.events
}

func (p *parser) appendFrag(frag Frag) {
	p.events = append(p.events, Event{
		Time: frag.Time,
		Type: TypeFrag,
		Frag: &frag,
	})
}

func (p *parser) appendState(state PlayerState) {
	p.events = append(p.events, Event{
		Time:  state.Time,
		Type:  TypePlayerState,
		State: &state,
	})
}

func (p *parser) appendShot(shot Shot) {
	p.events = append(p.events, Event{
		Time: shot.Time,
		Type: TypeShot,
		Shot: &shot,
	})
}

func (p *parser) appendKTXEvent(ktxEvent KTXEvent) {
	p.events = append(p.events, Event{
		Time: ktxEvent.Time,
		Type: TypeKTX,
		KTX:  &ktxEvent,
	})
}
