package item

import (
	"strings"

	"github.com/osm/quake/common/bsp"
	"github.com/osm/quake/protocol"
)

type Item struct {
	Short       string
	Long        string
	RespawnTime int
}

func FromBSPEntity(entity bsp.Entity) (Item, bool) {
	classname := strings.ToLower(strings.TrimSpace(entity.Classname))

	switch classname {
	case "weapon_super_shotgun":
		return Item{Short: "SSG", Long: "Super Shotgun", RespawnTime: 30}, true
	case "weapon_supernailgun":
		return Item{Short: "SNG", Long: "Super Nailgun", RespawnTime: 30}, true
	case "weapon_grenadelauncher":
		return Item{Short: "GL", Long: "Grenade Launcher", RespawnTime: 30}, true
	case "weapon_rocketlauncher":
		return Item{Short: "RL", Long: "Rocket Launcher", RespawnTime: 30}, true
	case "weapon_lightning":
		return Item{Short: "LG", Long: "Lightning Gun", RespawnTime: 30}, true
	case "item_armorinv":
		return Item{Short: "RA", Long: "Red Armor", RespawnTime: 20}, true
	case "item_armor2":
		return Item{Short: "YA", Long: "Yellow Armor", RespawnTime: 20}, true
	case "item_armor1":
		return Item{Short: "GA", Long: "Green Armor", RespawnTime: 20}, true
	case "item_artifact_super_damage":
		return Item{Short: "QUAD", Long: "Quad Damage", RespawnTime: 60}, true
	case "item_artifact_invulnerability":
		return Item{Short: "PENT", Long: "Pentagram of Protection", RespawnTime: 300}, true
	case "item_artifact_invisibility":
		return Item{Short: "RING", Long: "Ring of Shadows", RespawnTime: 300}, true
	case "item_health_mega":
		return Item{Short: "MH", Long: "Mega Health", RespawnTime: 20}, true
	}

	if classname == "item_health" &&
		strings.TrimSpace(entity.Value("spawnflags")) == "2" {
		return Item{Short: "MH", Long: "Mega Health", RespawnTime: 20}, true
	}

	return Item{}, false
}

func FromActiveWeapon(activeWeapon int32) (Item, bool) {
	switch activeWeapon {
	case protocol.ITShotgun:
		return Item{Short: "SG", Long: "Shotgun"}, true
	case protocol.ITSuperShotgun:
		return Item{Short: "SSG", Long: "Super Shotgun"}, true
	case protocol.ITNailgun:
		return Item{Short: "NG", Long: "Nailgun"}, true
	case protocol.ITSuperNailgun:
		return Item{Short: "SNG", Long: "Super Nailgun"}, true
	case protocol.ITGrenadeLauncher:
		return Item{Short: "GL", Long: "Grenade Launcher"}, true
	case protocol.ITRocketLauncher:
		return Item{Short: "RL", Long: "Rocket Launcher"}, true
	case protocol.ITLightning:
		return Item{Short: "LG", Long: "Lightning Gun"}, true
	case protocol.ITAx:
		return Item{Short: "AXE", Long: "Axe"}, true
	default:
		return Item{}, false
	}
}

func FromShort(short string) (Item, bool) {
	switch strings.ToUpper(strings.TrimSpace(short)) {
	case "SG":
		return Item{Short: "SG", Long: "Shotgun"}, true
	case "SSG":
		return Item{Short: "SSG", Long: "Super Shotgun"}, true
	case "NG":
		return Item{Short: "NG", Long: "Nailgun"}, true
	case "SNG":
		return Item{Short: "SNG", Long: "Super Nailgun"}, true
	case "GL":
		return Item{Short: "GL", Long: "Grenade Launcher"}, true
	case "RL":
		return Item{Short: "RL", Long: "Rocket Launcher"}, true
	case "LG":
		return Item{Short: "LG", Long: "Lightning Gun"}, true
	case "AXE":
		return Item{Short: "AXE", Long: "Axe"}, true
	case "MH":
		return Item{Short: "MH", Long: "Mega Health", RespawnTime: 20}, true
	case "RA":
		return Item{Short: "RA", Long: "Red Armor", RespawnTime: 20}, true
	case "YA":
		return Item{Short: "YA", Long: "Yellow Armor", RespawnTime: 20}, true
	case "GA":
		return Item{Short: "GA", Long: "Green Armor", RespawnTime: 20}, true
	case "QUAD":
		return Item{Short: "QUAD", Long: "Quad Damage", RespawnTime: 60}, true
	case "PENT":
		return Item{Short: "PENT", Long: "Pentagram of Protection", RespawnTime: 300}, true
	case "RING":
		return Item{Short: "RING", Long: "Ring of Shadows", RespawnTime: 300}, true
	default:
		return Item{}, false
	}
}
