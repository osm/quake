package standard

import (
	"encoding/binary"
	"testing"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/state"
)

func TestTrackerTracksCompletePlayerWireLayout(t *testing.T) {
	st := state.NewPacket(0)
	tracker := NewTracker(st)
	tracker.ctx.SetProtocolVersion(protocol.VersionQW)

	flags := uint16(
		protocol.PFMsec |
			protocol.PFCommand |
			protocol.PFVelocity1 |
			protocol.PFVelocity2 |
			protocol.PFVelocity3 |
			protocol.PFModel |
			protocol.PFSkinNum |
			protocol.PFEffects |
			protocol.PFWeaponFrame |
			protocol.PFDead,
	)
	extra := byte(
		protocol.CMAngle1 |
			protocol.CMAngle2 |
			protocol.CMAngle3 |
			protocol.CMForward |
			protocol.CMSide |
			protocol.CMUp |
			protocol.CMButtons |
			protocol.CMImpulse,
	)

	packet := make([]byte, protocol.QWServerPacketHeaderSize)
	packet = append(packet, protocol.SVCPlayerInfo, 3)
	packet = binary.LittleEndian.AppendUint16(packet, flags)
	packet = append(packet, 1, 2, 3, 4, 5, 6, 7)
	packet = append(packet, 0x44, extra)
	for _, value := range []uint16{0x1111, 0x2222, 0x3333, 0x4444, 0x5555, 0x6666} {
		packet = binary.LittleEndian.AppendUint16(packet, value)
	}
	packet = append(packet, 0x77, 0x88, 0x99)
	for _, value := range []uint16{0xaaaa, 0xbbbb, 0xcccc} {
		packet = binary.LittleEndian.AppendUint16(packet, value)
	}
	packet = append(packet, 4, 5, 6, 7, protocol.SVCNOP)

	if err := tracker.Observe(packet, 1); err != nil {
		t.Fatalf("decode player packet: %v", err)
	}
	if len(st.CurrentPlayers) != 1 {
		t.Fatalf("current player count = %d, want 1", len(st.CurrentPlayers))
	}
	record := state.PlayerRecordBytesLE(st.CurrentPlayers[0])
	if got := record[11]; got != 3 {
		t.Fatalf("player = %#x, want 0x03", got)
	}
	if got := record[46]; got != 0x44 {
		t.Fatalf("msec = %#x, want 0x44", got)
	}
	for _, check := range []struct {
		name string
		off  int
		want uint16
	}{
		{"angle 1", 14, 0x1111},
		{"angle 2", 12, 0x2222},
		{"forward", 20, 0x4444},
		{"side", 22, 0x5555},
		{"up", 18, 0x6666},
		{"velocity 1", 28, 0xaaaa},
		{"velocity 2", 30, 0xbbbb},
		{"velocity 3", 32, 0xcccc},
	} {
		if got := state.PlayerRecordUint16(&record, check.off); got != check.want {
			t.Fatalf("%s = %#x, want %#x", check.name, got, check.want)
		}
	}
	for _, check := range []struct {
		name string
		off  int
		want byte
	}{
		{"buttons", 17, 0x77},
		{"impulse", 16, 0x88},
		{"command msec", 24, 0x99},
		{"model", 25, 4},
		{"skin number", 26, 5},
		{"effects", 27, 6},
		{"weapon frame", 34, 7},
	} {
		if got := record[check.off]; got != check.want {
			t.Fatalf("%s = %#x, want %#x", check.name, got, check.want)
		}
	}
	if record[3]&0x80 == 0 {
		t.Fatal("dead flag was not retained")
	}
}
