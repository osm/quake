package quake

import (
	"net"
	"strings"
	"testing"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/updatestat"
	"github.com/osm/quake/packet/command/updatestatlong"
	"github.com/osm/quake/packet/command/updateuserinfo"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/server"
)

func TestLobbyStats(t *testing.T) {
	stats := LobbyStats{Health: 291, Armor: -1, Items: protocol.ITShotgun | protocol.ITArmor1, ActiveWeapon: protocol.ITShotgun}
	lobby := Lobby{Stats: func(server.Client) LobbyStats { return stats }}
	for _, health := range []int32{291, 100, 0} {
		stats.Health = health
		wire := (&svc.GameData{Seq: 1, Commands: lobby.stats(nil)}).Bytes()
		ctx := context.New(context.WithProtocolVersion(protocol.VersionQW))
		pkt, err := svc.Parse(ctx, wire)
		if err != nil {
			t.Fatal(err)
		}

		values := make(map[byte]int32)
		for _, raw := range pkt.(*svc.GameData).Commands {
			switch cmd := raw.(type) {
			case *updatestat.Command:
				values[cmd.Stat] = int32(cmd.Value8)
			case *updatestatlong.Command:
				values[cmd.Stat] = cmd.Value
			}
		}
		if len(values) != 9 || values[protocol.StatHealth] != health || values[protocol.StatArmor] != -1 {
			t.Fatalf("HUD counters changed on the wire: %v", values)
		}
		if uint32(values[protocol.StatItems]) != stats.Items || uint32(values[protocol.StatActiveWeapon]) != stats.ActiveWeapon {
			t.Fatalf("HUD icons changed on the wire: %v", values)
		}
	}
}

func TestLobbyPlayers(t *testing.T) {
	s := New(nil)
	if err := s.EnableLobby(Lobby{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < protocol.QWMaxClients; i++ {
		c := &client{addr: &net.UDPAddr{Port: i + 1}, done: make(chan struct{})}
		if !s.admitLobbyClient(c, "alice") {
			t.Fatal("lobby filled before reaching the player limit")
		}
		s.clients[c.GetAddr()] = c
	}
	if s.admitLobbyClient(&client{}, "extra") {
		t.Fatal("lobby exceeded the player limit")
	}
	alice := s.lookup(":1")
	bob := s.lookup(":2")
	if alice.slot == bob.slot || alice.GetName() == bob.GetName() {
		t.Fatal("clients share a slot or name")
	}
	if got := s.renameLobbyClient(bob, "ALICE"); got != "(2)ALICE" {
		t.Fatalf("duplicate rename = %q", got)
	}

	players := s.lobbyPlayers()
	for i := 0; i < 4; i++ {
		alice.updateScoreboard(players)
	}
	if alice.scoreboard != players || len(alice.updateScoreboard(players)) != 0 {
		t.Fatal("scoreboard did not converge")
	}
	s.removeClient(bob)
	commands := alice.updateScoreboard(s.lobbyPlayers())
	if len(commands) != 1 {
		t.Fatalf("disconnect sent %d scoreboard updates", len(commands))
	}
	removed := commands[0].(*updateuserinfo.Command)
	if removed.PlayerIndex != bob.slot || removed.UserID != 0 || removed.UserInfo != "" {
		t.Fatalf("disconnect did not clear the slot: %+v", removed)
	}
	replacement := &client{}
	if !s.admitLobbyClient(replacement, "bob") || replacement.slot != bob.slot || replacement.userID == bob.userID {
		t.Fatal("slot reuse did not assign a new identity")
	}
}

func TestLobbyNames(t *testing.T) {
	var players [protocol.QWMaxClients]lobbyPlayer
	players[0] = lobbyPlayer{name: "Alice", userID: 1}
	for _, test := range []struct{ input, want string }{
		{"alice", "(2)alice"},
		{" ALICE ", "(2)ALICE"},
		{"\xc1lice", "(2)\xc1lice"},
		{"\xff\n", "unnamed"},
		{"\xa0bob\xa0", "bob"},
		{strings.Repeat("a", 40), strings.Repeat("a", lobbyNameLength)},
	} {
		if got := uniqueLobbyName(test.input, players); got != test.want {
			t.Fatalf("name %q = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestLobbyTitle(t *testing.T) {
	s := New(nil)
	for _, test := range []struct {
		title string
		valid bool
	}{
		{"\xcde\xeeu", true},
		{"bad\xff", false},
		{"bad\x00", false},
		{"bad\n", false},
	} {
		if err := s.EnableLobby(Lobby{Title: test.title}); (err == nil) != test.valid {
			t.Fatalf("title %q: %v", test.title, err)
		}
	}
}
