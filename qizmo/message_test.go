package qizmo_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/state"
)

func TestLinkEncoderRoundTrip(t *testing.T) {
	packet := make([]byte, 0, 20)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet,
		protocol.SVCPlayerInfo, 0x00, 0x00, 0x00,
		0x34, 0x12, 0x78, 0x56, 0xbc, 0x9a, 0xde,
		protocol.SVCNOP,
	)
	assertPacketRoundTrip(t, packet)
}

func TestLinkEncoderOperationsRoundTrip(t *testing.T) {
	packet := binary.LittleEndian.AppendUint32(nil, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet,
		// Primary player.
		protocol.SVCPlayerInfo, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x80, 0x00, 0xc0, 0x00, 0x04,
		// Additional player with every semantically supported optional field.
		protocol.SVCPlayerInfo, 0x01, 0xff, 0x01,
		0x50, 0x00, 0x8c, 0x00, 0xc8, 0x00, 0x05,
		0x0a, 0xdd,
		0x11, 0x11, 0x22, 0x22, 0x33, 0x33, 0x44, 0x44,
		0x55, 0x55, 0x66, 0x77,
		0x01, 0x01, 0x02, 0x02, 0x03, 0x03,
		0x04, 0x05, 0x06, 0x07,
		// Sound from entity 5 at explicit coordinates.
		protocol.SVCSound, 0x2a, 0x00, 0x03,
		0xf4, 0x01, 0x58, 0x02, 0xbc, 0x02,
		// Damage and a point temporary entity.
		protocol.SVCDamage, 0x01, 0x02, 0x41, 0x00, 0x82, 0x00, 0xc3, 0x00,
		protocol.SVCTempEntity, protocol.TESuperSpike, 0xfe, 0x01, 0x62, 0x02, 0xc6, 0x02,
		// Raw-symbol string encodings.
		protocol.SVCPrint, protocol.PrintHigh, 'h', 'i', 0,
		protocol.SVCStuffText, 'x', 0,
	)
	packet = append(packet, protocol.SVCCenterPrint)
	packet = append(packet, assets.CenterPrintStrings[260]...)
	packet = append(packet,
		0,
		protocol.SVCSetInfo, 0x01, 'k', 0, 'v', 0,
		// One new packet entity with a model update.
		protocol.SVCPacketEntities, 0x21, 0x80, 0x04, 0x03, 0x00, 0x00,
	)
	assertPacketRoundTrip(t, packet)
}

func TestDecodePacketTracksPlayerName(t *testing.T) {
	ft := testFrequencyTables(t)
	packet := binary.LittleEndian.AppendUint32(nil, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet,
		protocol.SVCPlayerInfo, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 1,
		protocol.SVCUpdateUserInfo, 2, 1, 0, 0, 0,
	)
	packet = append(packet, []byte(`\team\red\name\ToT_slime`)...)
	packet = append(packet, 0)

	plan, ok := qizmo.NewLinkEncoder(ft, newTestPacketState()).PlanPacket(packet, 0)
	if !ok {
		t.Fatal("packet was not supported")
	}
	enc := rangeenc.New()
	if err := plan.EncodeBody(enc); err != nil {
		t.Fatalf("encode packet: %v", err)
	}
	dec := rangedec.NewPadded(enc.Finish())
	decodeState := newTestPacketState()
	if _, err := qizmo.NewDecoder(dec, ft, decodeState).DecodePacket(1, 1); err != nil {
		t.Fatalf("decode packet: %v", err)
	}
	if got := string(decodeState.PlayerName(2)); got != "ToT_slime" {
		t.Fatalf("decoded player name = %q, want ToT_slime", got)
	}
}

func assertPacketRoundTrip(t *testing.T, packet []byte) {
	t.Helper()
	ft := testFrequencyTables(t)

	encodeState := newTestPacketState()
	enc := rangeenc.New()
	plan, ok := qizmo.NewLinkEncoder(ft, encodeState).PlanPacket(packet, 0)
	if !ok {
		t.Fatal("packet was not supported")
	}
	if err := plan.EncodeBody(enc); err != nil {
		t.Fatalf("encode packet: %v", err)
	}

	dec := rangedec.NewPadded(enc.Finish())
	message, err := qizmo.NewDecoder(dec, ft, newTestPacketState()).DecodePacket(1, 1)
	if err != nil {
		t.Fatalf("decode packet: %v", err)
	}
	if !bytes.Equal(message, packet) {
		t.Fatalf("packet mismatch\ngot  %x\nwant %x", message, packet)
	}
}

func newTestPacketState() *state.Packet {
	return state.NewPacketWithAssets(0, assets.Embedded())
}

func TestLinkEncoderRejectsUnsupportedPacket(t *testing.T) {
	plan, ok := qizmo.NewLinkEncoder(testFrequencyTables(t), state.NewPacket(0)).PlanPacket([]byte{1, 2, 3}, 0)
	if ok {
		t.Fatal("unsupported packet reported as supported")
	}
	if plan != nil {
		t.Fatal("unsupported packet returned a plan")
	}
}

func testFrequencyTables(t *testing.T) *freq.Tables {
	t.Helper()
	tables, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("new frequency tables: %v", err)
	}
	return tables
}
