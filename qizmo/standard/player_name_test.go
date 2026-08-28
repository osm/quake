package standard

import (
	"testing"

	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/setinfo"
	"github.com/osm/quake/packet/command/updatename"
	"github.com/osm/quake/packet/command/updateuserinfo"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/state"
)

const testPlayer = 3

func TestTrackerTracksPlayerNames(t *testing.T) {
	st := state.NewPacket(0)
	tracker := NewTracker(st)
	tracker.ctx.SetProtocolVersion(protocol.VersionQW)

	observeNameCommands(t, tracker, 1, &updateuserinfo.Command{
		PlayerIndex: testPlayer,
		UserInfo:    `\team\red\name\ToT_slime`,
	})
	requirePlayerName(t, st, "ToT_slime")

	observeNameCommands(t, tracker, 2, &setinfo.Command{
		PlayerIndex: testPlayer,
		Key:         "name",
		Value:       "renamed",
	})
	requirePlayerName(t, st, "renamed")

	observeNameCommands(t, tracker, 3, &updatename.Command{
		PlayerIndex: testPlayer,
		Name:        "legacy",
	})
	requirePlayerName(t, st, "legacy")

	observeNameCommands(t, tracker, 4, &updateuserinfo.Command{PlayerIndex: testPlayer})
	requirePlayerName(t, st, "unnamed")
}

func observeNameCommands(t *testing.T, tracker *Tracker, sequence uint32, commands ...command.Command) {
	t.Helper()
	packet := make([]byte, protocol.QWServerPacketHeaderSize)
	for _, cmd := range commands {
		packet = append(packet, cmd.Bytes()...)
	}
	if err := tracker.Observe(packet, sequence); err != nil {
		t.Fatalf("observe sequence %d: %v", sequence, err)
	}
}

func requirePlayerName(t *testing.T, st *state.Packet, want string) {
	t.Helper()
	if got := string(st.PlayerName(testPlayer)); got != want {
		t.Fatalf("player %d name = %q, want %q", testPlayer, got, want)
	}
}
