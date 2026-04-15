package event

import (
	"strconv"
	"strings"

	"github.com/osm/quake/common/args"
	"github.com/osm/quake/common/ascii"
	"github.com/osm/quake/common/death"
	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/common/item"
	"github.com/osm/quake/packet/command/playerinfo"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/command/updatefrags"
	"github.com/osm/quake/packet/command/updatestat"
	"github.com/osm/quake/packet/command/updatestatlong"
	"github.com/osm/quake/packet/command/updateuserinfo"
	"github.com/osm/quake/protocol"
)

func (p *parser) handleUpdateUserInfo(cmd *updateuserinfo.Command) {
	info := infostring.Parse(cmd.UserInfo)
	name := strings.TrimSpace(ascii.Parse(info.Get("name")))
	if name == "" {
		return
	}

	p.playerTeams[cmd.PlayerIndex] = strings.TrimSpace(ascii.Parse(info.Get("team")))

	if oldName, ok := p.playerNames[cmd.PlayerIndex]; ok {
		delete(p.slotsByName, normalizeName(oldName))
	}

	p.playerNames[cmd.PlayerIndex] = name
	p.slotsByName[normalizeName(name)] = cmd.PlayerIndex
}

func (p *parser) handleStuffText(cmd *stufftext.Command) {
	for _, command := range args.Parse(cmd.String) {
		if command.Cmd == "fullserverinfo" && len(command.Args) > 0 {
			p.handleFullServerInfo(command.Args[0])
		}

		p.handleKTXStuffText(command)
	}
}

func (p *parser) handleFullServerInfo(infoString string) {
	info := infostring.Parse(infoString)

	mapName := strings.TrimSpace(info.Get("map"))
	if mapName != "" {
		p.mapName = mapName
	}

	value := strings.TrimSpace(info.Get("maxclients"))
	if value == "" {
		return
	}

	maxClients, err := strconv.Atoi(value)
	if err == nil && maxClients > 0 {
		p.maxClients = maxClients
	}
}

func (p *parser) handlePlayerInfo(cmd *playerinfo.Command) {
	if cmd == nil {
		return
	}

	currentPosition := p.positions[cmd.Index]
	positionChanged := false
	anglesChanged := false

	if cmd.IsMVD && cmd.MVD != nil {
		previousWeaponFrame := p.weaponFrames[cmd.Index]

		for axis := 0; axis < 3; axis++ {
			if cmd.MVD.Bits&(protocol.DFOrigin<<axis) == 0 {
				continue
			}

			positionChanged = true
			switch axis {
			case 0:
				currentPosition.X = cmd.MVD.Coord[0]
			case 1:
				currentPosition.Y = cmd.MVD.Coord[1]
			case 2:
				currentPosition.Z = cmd.MVD.Coord[2]
			}
		}

		currentAngles := p.viewAngles[cmd.Index]
		for axis := 0; axis < 3; axis++ {
			if cmd.MVD.Bits&(protocol.DFAngles<<axis) == 0 {
				continue
			}

			anglesChanged = true
			switch axis {
			case 0:
				currentAngles.X = cmd.MVD.Angle[0]
			case 1:
				currentAngles.Y = cmd.MVD.Angle[1]
			case 2:
				currentAngles.Z = cmd.MVD.Angle[2]
			}
		}

		if cmd.MVD.Bits&protocol.DFWeaponFrame != 0 {
			p.weaponFrames[cmd.Index] = cmd.MVD.WeaponFrame
			p.appendShotFromWeaponFrame(
				cmd.Index,
				currentPosition,
				currentAngles,
				previousWeaponFrame,
				cmd.MVD.WeaponFrame,
			)
		}

		p.positions[cmd.Index] = currentPosition
		p.viewAngles[cmd.Index] = currentAngles
		if positionChanged {
			p.recordPositionSample(cmd.Index, currentPosition)
		} else if anglesChanged {
			p.recordCurrentState(cmd.Index)
		}
		return
	}

	if cmd.Default == nil {
		return
	}

	currentPosition = Vec3{
		X: cmd.Default.Coord[0],
		Y: cmd.Default.Coord[1],
		Z: cmd.Default.Coord[2],
	}

	if userCommand := cmd.Default.DeltaUserCommand; userCommand != nil {
		p.viewAngles[cmd.Index] = Vec3{
			X: userCommand.CMAngle1,
			Y: userCommand.CMAngle2,
			Z: userCommand.CMAngle3,
		}
	}

	p.weaponFrames[cmd.Index] = cmd.Default.WeaponFrame
	p.positions[cmd.Index] = currentPosition
	p.recordPositionSample(cmd.Index, currentPosition)
}

func (p *parser) handleUpdateFrags(cmd *updatefrags.Command) {
	if cmd == nil {
		return
	}

	p.fragsBySlot[cmd.PlayerIndex] = int(cmd.Frags)
	p.recordCurrentState(cmd.PlayerIndex)
}

func (p *parser) handleUpdateStat(cmd *updatestat.Command) {
	if cmd == nil {
		return
	}

	value := int(cmd.Value8)
	statsChanged := false

	switch cmd.Stat {
	case protocol.StatHealth:
		p.healthBySlot[p.packetSlot] = value
		statsChanged = true
	case protocol.StatArmor:
		p.armorBySlot[p.packetSlot] = value
		statsChanged = true
	case protocol.StatActiveWeapon:
		p.activeWeapon[p.packetSlot] = int32(value)
		statsChanged = true
	}

	if statsChanged {
		p.recordCurrentState(p.packetSlot)
	}
}

func (p *parser) handleUpdateStatLong(cmd *updatestatlong.Command) {
	if cmd == nil {
		return
	}

	statsChanged := false

	switch cmd.Stat {
	case protocol.StatItems:
		p.itemFlags[p.packetSlot] = cmd.Value
		statsChanged = true
	case protocol.StatActiveWeapon:
		p.activeWeapon[p.packetSlot] = cmd.Value
		statsChanged = true
	case protocol.StatHealth:
		p.healthBySlot[p.packetSlot] = int(cmd.Value)
		statsChanged = true
	case protocol.StatArmor:
		p.armorBySlot[p.packetSlot] = int(cmd.Value)
		statsChanged = true
	}

	if statsChanged {
		p.recordCurrentState(p.packetSlot)
	}
}

func (p *parser) handlePrint(cmd *print.Command) {
	text := strings.TrimSpace(ascii.Parse(cmd.String))
	obituary, ok := death.Parse(text)
	if !ok || obituary == nil || obituary.Victim == "" {
		return
	}

	slot, ok := p.slotsByName[normalizeName(obituary.Victim)]
	if !ok {
		return
	}

	position, ok := p.positions[slot]
	if !ok {
		return
	}

	weapon, _ := item.FromShort(string(obituary.Weapon))
	frag := Frag{
		Time:   p.elapsed,
		Victim: obituary.Victim,
		Killer: obituary.Killer,
		Weapon: weapon,
		Pos:    position,
	}
	p.appendFrag(frag)
}
