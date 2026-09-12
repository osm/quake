package quake

import (
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/updatestat"
	"github.com/osm/quake/packet/command/updatestatlong"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/server"
)

type LobbyStats struct {
	Health       int32
	Armor        int32
	Ammo         int32
	Shells       int32
	Nails        int32
	Rockets      int32
	Cells        int32
	Items        uint32
	ActiveWeapon uint32
}

func (l *Lobby) stats(c server.Client) []command.Command {
	stats := LobbyStats{Health: 100}
	if l.Stats != nil {
		stats = l.Stats(c)
	}

	return []command.Command{
		lobbyStat(protocol.StatHealth, stats.Health),
		lobbyStat(protocol.StatArmor, stats.Armor),
		lobbyStat(protocol.StatAmmo, stats.Ammo),
		lobbyStat(protocol.StatShells, stats.Shells),
		lobbyStat(protocol.StatNails, stats.Nails),
		lobbyStat(protocol.StatRockets, stats.Rockets),
		lobbyStat(protocol.StatCells, stats.Cells),
		lobbyStat(protocol.StatItems, int32(stats.Items)),
		lobbyStat(protocol.StatActiveWeapon, int32(stats.ActiveWeapon)),
	}
}

func lobbyStat(stat byte, value int32) command.Command {
	if value >= 0 && value <= 255 {
		return &updatestat.Command{Stat: stat, Value8: byte(value)}
	}

	return &updatestatlong.Command{Stat: stat, Value: value}
}
