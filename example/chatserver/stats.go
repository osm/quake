package main

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/server"
	"github.com/osm/quake/server/quake"
)

func menuStats(server.Client) quake.LobbyStats {
	return quake.LobbyStats{
		Health:       100,
		Ammo:         100,
		Armor:        100,
		Items:        protocol.ITArmor1 | protocol.ITShotgun | protocol.ITShells,
		ActiveWeapon: protocol.ITShotgun,
	}
}
