package death

import "strings"

type templateType uint8

const (
	PlayerDeath templateType = iota
	PlayerSuicide
	XFraggedByY
	XFragsY
	XTeamKilled
	XTeamKills
)

func toObituaryType(t templateType) Type {
	switch t {
	case PlayerDeath:
		return Death
	case PlayerSuicide:
		return Suicide
	case XFraggedByY, XFragsY:
		return Kill
	case XTeamKilled:
		return TeamKillUnknownKiller
	case XTeamKills:
		return TeamKillUnknownVictim
	default:
		return Unknown
	}
}

type template struct {
	parser   parser
	parts    []string
	priority int
	typ      templateType
	weapon   Weapon
}

func newTemplate(typ templateType, weapon Weapon, parts ...string) template {
	prio := 0
	for _, pr := range parts {
		prio += len(pr)
	}

	var pa parser
	switch typ {
	case XFraggedByY, XFragsY:
		pa = infixParser{
			sep:    parts[0],
			suffix: strings.Join(parts[1:], ""),
		}
	default:
		pa = suffixParser{
			suffix: strings.Join(parts, ""),
		}
	}

	return template{
		parser:   pa,
		parts:    parts,
		priority: prio,
		typ:      typ,
		weapon:   weapon,
	}
}

var templates = []template{
	newTemplate(PlayerDeath, Drown, " sleeps with the fishes"),
	newTemplate(PlayerDeath, Drown, " sucks it down"),
	newTemplate(PlayerDeath, Fall, " cratered"),
	newTemplate(PlayerDeath, Fall, " fell to her death"),
	newTemplate(PlayerDeath, Fall, " fell to his death"),
	newTemplate(PlayerDeath, Lava, " burst into flames"),
	newTemplate(PlayerDeath, Lava, " turned into hot slag"),
	newTemplate(PlayerDeath, Lava, " visits the Volcano God"),
	newTemplate(PlayerDeath, NoWeapon, " died"),
	newTemplate(PlayerDeath, NoWeapon, " tried to leave"),
	newTemplate(PlayerDeath, Slime, " can't exist on slime alone"),
	newTemplate(PlayerDeath, Slime, " gulped a load of slime"),
	newTemplate(PlayerDeath, Squish, " was squished"),
	newTemplate(PlayerDeath, Trap, " ate a lavaball"),
	newTemplate(PlayerDeath, Trap, " blew up"),
	newTemplate(PlayerDeath, Trap, " was spiked"),
	newTemplate(PlayerDeath, Trap, " was zapped"),
	newTemplate(PlayerSuicide, Discharge, " discharges into the lava"),
	newTemplate(PlayerSuicide, Discharge, " discharges into the slime"),
	newTemplate(PlayerSuicide, Discharge, " discharges into the water"),
	newTemplate(PlayerSuicide, Discharge, " electrocutes herself"),
	newTemplate(PlayerSuicide, Discharge, " electrocutes himself"),
	newTemplate(PlayerSuicide, Discharge, " heats up the water"),
	newTemplate(PlayerSuicide, Discharge, " railcutes herself"),
	newTemplate(PlayerSuicide, Discharge, " railcutes himself"),
	newTemplate(PlayerSuicide, GrenadeLauncher, " tries to put the pin back in"),
	newTemplate(PlayerSuicide, NoWeapon, " suicides"),
	newTemplate(PlayerSuicide, RocketLauncher, " becomes bored with life"),
	newTemplate(PlayerSuicide, RocketLauncher, " discovers blast radius"),
	newTemplate(XFraggedByY, "Axe", " was ax-murdered by "),
	newTemplate(XFraggedByY, "Axe", " was axed to pieces by "),
	newTemplate(XFraggedByY, QShotgun, " was lead poisoned by "),
	newTemplate(XFraggedByY, QSuperNailgun, " was straw-cuttered by "),
	newTemplate(XFraggedByY, Discharge, " accepts ", "' discharge"),
	newTemplate(XFraggedByY, Discharge, " accepts ", "'s discharge"),
	newTemplate(XFraggedByY, Discharge, " drains ", "' batteries"),
	newTemplate(XFraggedByY, Discharge, " drains ", "'s batteries"),
	newTemplate(XFraggedByY, GrenadeLauncher, " eats ", "' pineapple"),
	newTemplate(XFraggedByY, GrenadeLauncher, " eats ", "'s pineapple"),
	newTemplate(XFraggedByY, GrenadeLauncher, " was gibbed by ", "' grenade"),
	newTemplate(XFraggedByY, GrenadeLauncher, " was gibbed by ", "'s grenade"),
	newTemplate(XFraggedByY, LightningGun, " accepts ", "' shaft"),
	newTemplate(XFraggedByY, LightningGun, " accepts ", "'s shaft"),
	newTemplate(XFraggedByY, Nailgun, " was body pierced by "),
	newTemplate(XFraggedByY, Nailgun, " was nailed by "),
	newTemplate(XFraggedByY, QLightningGun, " gets a natural disaster from "),
	newTemplate(XFraggedByY, QRocketLauncher, " was brutalized by ", "' quad rocket"),
	newTemplate(XFraggedByY, QRocketLauncher, " was brutalized by ", "'s quad rocket"),
	newTemplate(XFraggedByY, QRocketLauncher, " was smeared by ", "' quad rocket"),
	newTemplate(XFraggedByY, QRocketLauncher, " was smeared by ", "'s quad rocket"),
	newTemplate(XFraggedByY, QSuperShotgun, " ate 8 loads of ", "' buckshot"),
	newTemplate(XFraggedByY, QSuperShotgun, " ate 8 loads of ", "'s buckshot"),
	newTemplate(XFraggedByY, RocketLauncher, " rides ", "' rocket"),
	newTemplate(XFraggedByY, RocketLauncher, " rides ", "'s rocket"),
	newTemplate(XFraggedByY, RocketLauncher, " was gibbed by ", "' rocket"),
	newTemplate(XFraggedByY, RocketLauncher, " was gibbed by ", "'s rocket"),
	newTemplate(XFraggedByY, Shotgun, " chewed on ", "' boomstick"),
	newTemplate(XFraggedByY, Shotgun, " chewed on ", "'s boomstick"),
	newTemplate(XFraggedByY, Stomp, " softens ", "' fall"),
	newTemplate(XFraggedByY, Stomp, " softens ", "'s fall"),
	newTemplate(XFraggedByY, Stomp, " tried to catch "),
	newTemplate(XFraggedByY, Stomp, " was crushed by "),
	newTemplate(XFraggedByY, Stomp, " was jumped by "),
	newTemplate(XFraggedByY, Stomp, " was literally stomped into particles by "),
	newTemplate(XFraggedByY, SuperNailgun, " was perforated by "),
	newTemplate(XFraggedByY, SuperNailgun, " was punctured by "),
	newTemplate(XFraggedByY, SuperNailgun, " was ventilated by "),
	newTemplate(XFraggedByY, SuperShotgun, " ate 2 loads of ", "' buckshot"),
	newTemplate(XFraggedByY, SuperShotgun, " ate 2 loads of ", "'s buckshot"),
	newTemplate(XFraggedByY, Telefrag, " was telefragged by "),
	newTemplate(XFragsY, QRocketLauncher, " rips ", " a new one"),
	newTemplate(XFragsY, Squish, " squishes "),
	newTemplate(XFragsY, Stomp, " stomps "),
	newTemplate(XTeamKilled, Stomp, " was crushed by her teammate"),
	newTemplate(XTeamKilled, Stomp, " was crushed by his teammate"),
	newTemplate(XTeamKilled, Stomp, " was jumped by her teammate"),
	newTemplate(XTeamKilled, Stomp, " was jumped by his teammate"),
	newTemplate(XTeamKilled, Telefrag, " was telefragged by her teammate"),
	newTemplate(XTeamKilled, Telefrag, " was telefragged by his teammate"),
	newTemplate(XTeamKills, Squish, " squished a teammate"),
	newTemplate(XTeamKills, TeamKill, " checks her glasses"),
	newTemplate(XTeamKills, TeamKill, " checks his glasses"),
	newTemplate(XTeamKills, TeamKill, " gets a frag for the other team"),
	newTemplate(XTeamKills, TeamKill, " loses another friend"),
	newTemplate(XTeamKills, TeamKill, " mows down a teammate"),
}
