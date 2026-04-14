package item

import (
	"testing"

	"github.com/osm/quake/common/bsp"
	"github.com/osm/quake/protocol"
)

func TestFromBSPEntity(t *testing.T) {
	tests := []struct {
		name        string
		entity      bsp.Entity
		wantShort   string
		wantLong    string
		wantRespawn int
		ok          bool
	}{
		{
			name: "rocket launcher",
			entity: bsp.Entity{
				Classname: "weapon_rocketlauncher",
			},
			wantShort:   "RL",
			wantLong:    "Rocket Launcher",
			wantRespawn: 30,
			ok:          true,
		},
		{
			name: "red armor",
			entity: bsp.Entity{
				Classname: "item_armorinv",
			},
			wantShort:   "RA",
			wantLong:    "Red Armor",
			wantRespawn: 20,
			ok:          true,
		},
		{
			name: "mega health via spawnflags",
			entity: bsp.Entity{
				Classname: "item_health",
				Pairs: []bsp.EntityField{
					{Key: "spawnflags", Value: "2"},
				},
			},
			wantShort:   "MH",
			wantLong:    "Mega Health",
			wantRespawn: 20,
			ok:          true,
		},
		{
			name: "non item entity",
			entity: bsp.Entity{
				Classname: "info_player_start",
			},
			ok: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item, ok := FromBSPEntity(test.entity)
			if ok != test.ok {
				t.Fatalf("ok = %v, want %v", ok, test.ok)
			}
			if item.Short != test.wantShort {
				t.Fatalf("short = %q, want %q", item.Short, test.wantShort)
			}
			if item.Long != test.wantLong {
				t.Fatalf("long = %q, want %q", item.Long, test.wantLong)
			}
			if item.RespawnTime != test.wantRespawn {
				t.Fatalf(
					"respawn = %d, want %d",
					item.RespawnTime,
					test.wantRespawn,
				)
			}
		})
	}
}

func TestFromActiveWeapon(t *testing.T) {
	tests := []struct {
		name      string
		active    int32
		wantShort string
		wantLong  string
		wantOK    bool
	}{
		{
			name:      "shotgun",
			active:    protocol.ITShotgun,
			wantShort: "SG",
			wantLong:  "Shotgun",
			wantOK:    true,
		},
		{
			name:      "rocket launcher",
			active:    protocol.ITRocketLauncher,
			wantShort: "RL",
			wantLong:  "Rocket Launcher",
			wantOK:    true,
		},
		{
			name:      "axe",
			active:    protocol.ITAx,
			wantShort: "AXE",
			wantLong:  "Axe",
			wantOK:    true,
		},
		{
			name:   "unknown",
			active: 0,
			wantOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := FromActiveWeapon(test.active)
			if ok != test.wantOK {
				t.Fatalf("ok = %v, want %v", ok, test.wantOK)
			}
			if !test.wantOK {
				return
			}
			if got.Short != test.wantShort {
				t.Fatalf("short = %q, want %q", got.Short, test.wantShort)
			}
			if got.Long != test.wantLong {
				t.Fatalf("long = %q, want %q", got.Long, test.wantLong)
			}
		})
	}
}

func TestFromShort(t *testing.T) {
	tests := []struct {
		name      string
		short     string
		wantShort string
		wantLong  string
		wantOK    bool
	}{
		{
			name:      "rocket launcher",
			short:     "RL",
			wantShort: "RL",
			wantLong:  "Rocket Launcher",
			wantOK:    true,
		},
		{
			name:      "ring",
			short:     "RING",
			wantShort: "RING",
			wantLong:  "Ring of Shadows",
			wantOK:    true,
		},
		{
			name:   "unknown",
			short:  "BOGUS",
			wantOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := FromShort(test.short)
			if ok != test.wantOK {
				t.Fatalf("ok = %v, want %v", ok, test.wantOK)
			}
			if !test.wantOK {
				return
			}
			if got.Short != test.wantShort {
				t.Fatalf("short = %q, want %q", got.Short, test.wantShort)
			}
			if got.Long != test.wantLong {
				t.Fatalf("long = %q, want %q", got.Long, test.wantLong)
			}
		})
	}
}
