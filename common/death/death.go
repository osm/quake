package death

import (
	"fmt"
	"sort"
	"strings"
)

type Weapon string

const (
	Axe             Weapon = "axe"
	Discharge       Weapon = "discharge"
	Drown           Weapon = "drown"
	Fall            Weapon = "fall"
	GrenadeLauncher Weapon = "gl"
	Lava            Weapon = "lava"
	LightningGun    Weapon = "lg"
	Nailgun         Weapon = "ng"
	NoWeapon        Weapon = "no weapon"
	QLightningGun   Weapon = "quad lg"
	QRocketLauncher Weapon = "quad rl"
	QShotgun        Weapon = "quad sg"
	QSuperNailgun   Weapon = "quad sng"
	QSuperShotgun   Weapon = "quad ssg"
	RocketLauncher  Weapon = "rl"
	Shotgun         Weapon = "sg"
	Slime           Weapon = "slime"
	Squish          Weapon = "squish"
	Stomp           Weapon = "stomp"
	SuperNailgun    Weapon = "sng"
	SuperShotgun    Weapon = "ssg"
	TeamKill        Weapon = "tk"
	Telefrag        Weapon = "telefrag"
	Trap            Weapon = "trap"
)

type Type uint8

const (
	Unknown Type = iota
	Death
	Suicide
	Kill
	TeamKillUnknownKiller
	TeamKillUnknownVictim
)

func (t Type) String() string {
	names := []string{
		"unknown",
		"death",
		"suicide",
		"kill",
		"teamkill",
		"teamkill",
	}

	if t < 0 || int(t) >= len(names) {
		return "type(?)"
	}

	return names[t]
}

type Obituary struct {
	Type   Type
	Killer string
	Victim string
	Weapon Weapon
}

func (o Obituary) String() string {
	hasKiller := o.Killer != ""
	hasVictim := o.Victim != ""

	switch {
	case hasKiller && hasVictim:
		return fmt.Sprintf("%s %s %s %s", o.Type, o.Killer, o.Weapon, o.Victim)
	case hasKiller:
		return fmt.Sprintf("%s %s %s", o.Type, o.Killer, o.Weapon)
	default:
		return fmt.Sprintf("%s %s %s", o.Type, o.Victim, o.Weapon)
	}
}

type Parser struct {
	templates []template
}

func init() {
	sort.SliceStable(templates, func(i, j int) bool {
		return templates[i].priority > templates[j].priority
	})
}

func Parse(input string) (*Obituary, bool) {
	s := strings.TrimSpace(input)

	for _, t := range templates {
		x, y, ok := t.parser.parse(s)
		if !ok {
			continue
		}

		ob := &Obituary{Weapon: t.weapon, Type: toObituaryType(t.typ)}

		switch t.typ {
		case PlayerDeath:
			ob.Victim = x
		case PlayerSuicide:
			ob.Victim = x
		case XFraggedByY:
			ob.Victim = x
			ob.Killer = y
		case XFragsY:
			ob.Killer = x
			ob.Victim = y
		case XTeamKilled:
			ob.Victim = x
		case XTeamKills:
			ob.Killer = x
		default:
			ob.Victim = x
			ob.Killer = y
		}

		return ob, true
	}

	return nil, false
}
