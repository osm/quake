package event

import (
	"strconv"

	"github.com/osm/quake/common/args"
)

func (p *parser) handleKTXStuffText(command args.Arg) {
	if command.Cmd == "//ktx" &&
		len(command.Args) > 0 &&
		command.Args[0] == "matchstart" {
		p.handleKTXMatchStart()
		return
	}

	if command.Cmd != "//ktx" || len(command.Args) < 3 {
		return
	}

	p.handleKTXCommand(command.Args)
}

func (p *parser) handleKTXMatchStart() {
	p.sawMatchStart = true
	p.matchRunning = true
	p.elapsed = 0

	p.matchStartIndex = len(p.events)
	p.lastSample = make(map[byte]Vec3)
}

func (p *parser) handleKTXCommand(arguments []string) {
	switch arguments[0] {
	case "took", "timer":
		entity, err1 := strconv.Atoi(arguments[1])
		seconds, err2 := strconv.ParseFloat(arguments[2], 64)
		if err1 != nil || err2 != nil {
			return
		}

		playerEdict := 0
		if arguments[0] == "took" && len(arguments) >= 4 {
			playerEdict, _ = strconv.Atoi(arguments[3])
		}

		action := KTXActionTook
		if arguments[0] == "timer" {
			action = KTXActionTimer
		}

		p.appendKTXEvent(KTXEvent{
			Time:        p.elapsed,
			Entity:      entity,
			Seconds:     seconds,
			Action:      action,
			PlayerEdict: playerEdict,
		})

	case "drop":
		entity, err1 := strconv.Atoi(arguments[1])
		itemBits, err2 := strconv.Atoi(arguments[2])
		if err1 != nil || err2 != nil {
			return
		}

		playerEdict := 0
		if len(arguments) >= 4 {
			playerEdict, _ = strconv.Atoi(arguments[3])
		}

		p.appendKTXEvent(KTXEvent{
			Time:        p.elapsed,
			Entity:      entity,
			Action:      KTXActionDrop,
			ItemBits:    itemBits,
			PlayerEdict: playerEdict,
		})

	case "bp":
		entity, err := strconv.Atoi(arguments[1])
		if err != nil {
			return
		}

		playerEdict := 0
		if len(arguments) >= 3 {
			playerEdict, _ = strconv.Atoi(arguments[2])
		}

		p.appendKTXEvent(KTXEvent{
			Time:        p.elapsed,
			Entity:      entity,
			Action:      KTXActionBackpackPickup,
			PlayerEdict: playerEdict,
		})
	}
}
