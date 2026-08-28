package message

import (
	"testing"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/state"
)

func TestTrackPlayerIdentity(t *testing.T) {
	st := state.NewPacket(0)
	trackPlayerIdentity(st, protocol.SVCUpdateUserInfo, []byte{
		2, 1, 0, 0, 0,
		'\\', 'n', 'a', 'm', 'e', '\\', 's', 'l', 'i', 'm', 'e', 0,
	})
	if got := string(st.PlayerName(2)); got != "slime" {
		t.Fatalf("updateuserinfo name = %q, want slime", got)
	}

	trackPlayerIdentity(st, protocol.SVCSetInfo, []byte{
		2, 'n', 'a', 'm', 'e', 0, 'T', 'o', 'T', 0,
	})
	if got := string(st.PlayerName(2)); got != "ToT" {
		t.Fatalf("setinfo name = %q, want ToT", got)
	}
}
