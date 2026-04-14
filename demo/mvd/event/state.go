package event

import (
	"strings"

	"github.com/osm/quake/common/item"
	"github.com/osm/quake/protocol"
)

func (p *parser) recordPositionSample(slot byte, position Vec3) {
	if p.playerName(slot) == "" {
		return
	}

	if previous, ok := p.lastSample[slot]; ok && previous.Equal(position) {
		return
	}

	p.lastSample[slot] = position

	state := p.makePlayerState(slot, position)
	p.appendState(state)
}

func (p *parser) recordCurrentState(slot byte) {
	if p.playerName(slot) == "" {
		return
	}

	position, ok := p.positions[slot]
	if !ok {
		return
	}

	state := p.makePlayerState(slot, position)
	p.appendState(state)
}

func (p *parser) makePlayerState(slot byte, position Vec3) PlayerState {
	return PlayerState{
		Time:       p.elapsed,
		Edict:      p.playerEdict(slot),
		Player:     p.playerName(slot),
		Team:       p.playerTeams[slot],
		Pos:        position,
		ViewAngles: p.viewAngles[slot],
		Frags:      p.fragsBySlot[slot],
		Health:     p.healthBySlot[slot],
		Armor:      p.armorBySlot[slot],
		Weapon:     p.activeWeaponItem(slot),

		HasSG:   p.itemFlags[slot]&protocol.ITShotgun == protocol.ITShotgun,
		HasNG:   p.itemFlags[slot]&protocol.ITNailgun == protocol.ITNailgun,
		HasSSG:  p.itemFlags[slot]&protocol.ITSuperShotgun == protocol.ITSuperShotgun,
		HasSNG:  p.itemFlags[slot]&protocol.ITSuperNailgun == protocol.ITSuperNailgun,
		HasGL:   p.itemFlags[slot]&protocol.ITGrenadeLauncher == protocol.ITGrenadeLauncher,
		HasRL:   p.itemFlags[slot]&protocol.ITRocketLauncher == protocol.ITRocketLauncher,
		HasLG:   p.itemFlags[slot]&protocol.ITLightning == protocol.ITLightning,
		HasQuad: p.itemFlags[slot]&protocol.ITQuad == protocol.ITQuad,
		HasRing: p.itemFlags[slot]&protocol.ITInvisibility == protocol.ITInvisibility,
		HasPent: p.itemFlags[slot]&protocol.ITInvulnerability == protocol.ITInvulnerability,
	}
}

func (p *parser) appendShotFromWeaponFrame(
	slot byte,
	position Vec3,
	viewAngles Vec3,
	previousWeaponFrame byte,
	currentWeaponFrame byte,
) {
	if currentWeaponFrame != 1 || previousWeaponFrame == 1 {
		return
	}

	playerName := p.playerName(slot)
	if playerName == "" {
		return
	}

	p.appendShot(Shot{
		Time:       p.elapsed,
		Edict:      p.playerEdict(slot),
		Player:     playerName,
		Team:       p.playerTeams[slot],
		Pos:        position,
		ViewAngles: viewAngles,
		Weapon:     p.activeWeaponItem(slot),
	})
}

func (p *parser) playerName(slot byte) string {
	return strings.TrimSpace(p.playerNames[slot])
}

func (p *parser) playerEdict(slot byte) int {
	return int(slot) + 1
}

func (p *parser) activeWeaponItem(slot byte) item.Item {
	weapon, _ := item.FromActiveWeapon(p.activeWeapon[slot])
	return weapon
}
