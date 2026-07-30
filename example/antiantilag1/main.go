package main

import (
	"flag"
	"log"
	"os"
	"sync"
	"time"

	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command/deltausercommand"
	"github.com/osm/quake/packet/command/move"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/proxy"
)

const maxDemonstrationDelay = 80 * time.Millisecond
const impulseRocketLauncher = 7
const rocketAttackInterval = 800 * time.Millisecond

type fireRocketDetector struct {
	mu              sync.Mutex
	weaponImpulses  map[string]byte
	fireRocketTimes map[string]time.Time
}

func main() {
	listenAddr := flag.String(
		"listen-addr",
		"127.0.0.1:27500",
		"address on which the demonstration proxy listens",
	)
	delay := flag.Duration(
		"delay",
		80*time.Millisecond,
		"duration for which CLC traffic is held after an attack",
	)
	flag.Parse()

	logger := log.New(os.Stdout, "ANTI ANTILAG 1: ", log.Ldate|log.Ltime|log.Lmicroseconds)

	if *delay <= 0 || *delay > maxDemonstrationDelay {
		logger.Fatalf(
			"delay must be greater than zero and no more than %s",
			maxDemonstrationDelay,
		)
	}

	detector := fireRocketDetector{
		weaponImpulses:  make(map[string]byte),
		fireRocketTimes: make(map[string]time.Time),
	}
	prx := proxy.New(
		proxy.WithLogger(logger),
	)
	prx.HandleFunc(proxy.CLC, func(client *proxy.Client, pkt packet.Packet) {
		if !detector.fireRocket(client.Address(), pkt) {
			return
		}

		client.DelayCLCFor(*delay)
		logger.Printf(
			"rocket attack from %s; holding ordered CLC traffic for %s",
			client.Address(),
			*delay,
		)
	})

	logger.Printf("listening on %s", *listenAddr)
	if err := prx.Serve(*listenAddr); err != nil {
		logger.Fatalf("unable to serve, %v", err)
	}
}

func (d *fireRocketDetector) fireRocket(clientID string, pkt packet.Packet) bool {
	gameData, ok := pkt.(*clc.GameData)
	if !ok {
		return false
	}

	for _, raw := range gameData.Commands {
		moveCommand, ok := raw.(*move.Command)
		if !ok {
			continue
		}

		buttons, impulse := latestInput(moveCommand)

		d.mu.Lock()
		if isWeaponImpulse(impulse) {
			d.weaponImpulses[clientID] = impulse
		}
		weaponImpulse := d.weaponImpulses[clientID]
		lastFireRocket := d.fireRocketTimes[clientID]
		now := time.Now()
		fireRocket := shouldDelayFireRocket(buttons, weaponImpulse, lastFireRocket, now)
		if fireRocket {
			d.fireRocketTimes[clientID] = now
		}
		d.mu.Unlock()

		return fireRocket
	}

	return false
}

func shouldDelayFireRocket(
	buttons byte,
	weaponImpulse byte,
	lastFireRocket time.Time,
	now time.Time,
) bool {
	if buttons&protocol.ButtonAttack == 0 || weaponImpulse != impulseRocketLauncher {
		return false
	}
	return lastFireRocket.IsZero() || now.Sub(lastFireRocket) >= rocketAttackInterval
}

func latestInput(command *move.Command) (byte, byte) {
	var buttons byte
	var impulse byte
	for _, delta := range []*deltausercommand.Command{
		command.Null,
		command.Old,
		command.New,
	} {
		if delta != nil && delta.Bits&protocol.CMButtons != 0 {
			buttons = delta.CMButtons
		}
		if delta != nil && delta.Bits&protocol.CMImpulse != 0 && delta.CMImpulse != 0 {
			impulse = delta.CMImpulse
		}
	}
	return buttons, impulse
}

func isWeaponImpulse(impulse byte) bool {
	return impulse >= 1 && impulse <= 8
}
