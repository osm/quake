package qwz

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

func TestCanonicalizeDisconnectPacket(t *testing.T) {
	packet := binary.LittleEndian.AppendUint32(nil, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet, protocol.SVCNOP, protocol.SVCDisconnect, protocol.SVCNOP)

	got, disconnected := canonicalizeConnectionPacket(context.New(), packet, false)
	want := append([]byte(nil), packet...)
	want[protocol.QWServerPacketHeaderSize+1] = protocol.SVCNOP
	if !disconnected || !bytes.Equal(got, want) {
		t.Fatalf("canonical packet = %x disconnected = %v, want %x and true", got, disconnected, want)
	}
}

func TestCanonicalizeDisconnectedPacketDropsPlayerInfo(t *testing.T) {
	packet := binary.LittleEndian.AppendUint32(nil, 2)
	packet = binary.LittleEndian.AppendUint32(packet, 2)
	packet = append(packet,
		protocol.SVCPlayerInfo, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0,
		protocol.SVCNOP,
	)

	got, disconnected := canonicalizeConnectionPacket(context.New(), packet, true)
	want := append([]byte(nil), packet[:protocol.QWServerPacketHeaderSize]...)
	want = append(want, protocol.SVCNOP)
	if disconnected || !bytes.Equal(got, want) {
		t.Fatalf("canonical packet = %x disconnected = %v, want %x and false", got, disconnected, want)
	}
}
