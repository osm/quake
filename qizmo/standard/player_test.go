package standard

import (
	"encoding/binary"
	"testing"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/internal/wire"
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
	if got := record[wire.PlayerIndexOffset]; got != 3 {
		t.Fatalf("player = %#x, want 0x03", got)
	}
	if got := record[wire.PlayerMsecOffset]; got != 0x44 {
		t.Fatalf("msec = %#x, want 0x44", got)
	}
	for _, check := range []struct {
		name string
		off  int
		want uint16
	}{
		{"angle 1", wire.PlayerAngleOffset + 2, 0x1111},
		{"angle 2", wire.PlayerAngleOffset, 0x2222},
		{"forward", wire.PlayerMoveOffset, 0x4444},
		{"side", wire.PlayerMoveOffset + 2, 0x5555},
		{"up", wire.PlayerRollOffset, 0x6666},
		{"velocity 1", wire.PlayerVelocityOffset, 0xaaaa},
		{"velocity 2", wire.PlayerVelocityOffset + 2, 0xbbbb},
		{"velocity 3", wire.PlayerVelocityOffset + 4, 0xcccc},
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
		{"buttons", wire.PlayerButtonsOffset, 0x77},
		{"impulse", wire.PlayerImpulseOffset, 0x88},
		{"command msec", wire.PlayerCommandMsecOffset, 0x99},
		{"model", wire.PlayerModelOffset, 4},
		{"skin number", wire.PlayerSkinNumOffset, 5},
		{"effects", wire.PlayerEffectsOffset, 6},
		{"weapon frame", wire.PlayerWeaponFrameOffset, 7},
	} {
		if got := record[check.off]; got != check.want {
			t.Fatalf("%s = %#x, want %#x", check.name, got, check.want)
		}
	}
	if record[wire.PlayerMotionMaskOffset]&wire.PlayerDead == 0 {
		t.Fatal("dead flag was not retained")
	}
}
