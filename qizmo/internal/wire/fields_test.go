package wire

import (
	"testing"

	"github.com/osm/quake/protocol"
)

func TestPacketEntityFieldsFollowWireOrder(t *testing.T) {
	want := [...]PacketEntityField{
		{protocol.UModel, 4, 1},
		{protocol.UFrame, 5, 1},
		{protocol.UColorMap, 6, 1},
		{protocol.USkin, 7, 1},
		{protocol.UEffects, 8, 1},
		{protocol.UOrigin1, 12, 2},
		{protocol.UAngle1, 9, 1},
		{protocol.UOrigin2, 14, 2},
		{protocol.UAngle2, 10, 1},
		{protocol.UOrigin3, 16, 2},
		{protocol.UAngle3, 11, 1},
	}
	if PacketEntityFields != want {
		t.Fatalf("packet-entity fields = %#v, want %#v", PacketEntityFields, want)
	}
}

func TestPlayerCommandFieldsFollowWireOrder(t *testing.T) {
	want := [...]PlayerCommandField{
		{protocol.CMAngle1, 14, 2},
		{protocol.CMAngle2, 12, 2},
		{protocol.CMAngle3, UntrackedRecordOffset, 2},
		{protocol.CMForward, 20, 2},
		{protocol.CMSide, 22, 2},
		{protocol.CMUp, 18, 2},
		{protocol.CMButtons, 17, 1},
		{protocol.CMImpulse, 16, 1},
	}
	if PlayerCommandFields != want {
		t.Fatalf("player command fields = %#v, want %#v", PlayerCommandFields, want)
	}
}
