package message

import (
	"testing"

	"github.com/osm/quake/qizmo/state"
)

func TestDecodeChatSymbolPlayerName(t *testing.T) {
	st := state.NewPacket(0)
	st.SetPlayerName(3, "ToT_slime")

	for _, test := range []struct {
		symbol uint32
		chat   bool
		want   string
	}{
		{0x103, false, "ToT_slime"},
		{0x103, true, "(ToT_slime): "},
		{0x123, true, "ToT_slime"},
	} {
		got, err := decodePrintSymbol(st, test.symbol, test.chat)
		if err != nil {
			t.Fatalf("decode symbol %#x: %v", test.symbol, err)
		}
		if string(got) != test.want {
			t.Fatalf("symbol %#x = %q, want %q", test.symbol, got, test.want)
		}
	}
}
