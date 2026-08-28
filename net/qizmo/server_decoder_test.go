package qizmo_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/osm/quake/protocol"
	qizmocodec "github.com/osm/quake/qizmo"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/state"
)

const testCommandMsec = 13

func TestServerDecoderRawAndCompressed(t *testing.T) {
	frequencies, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("load frequency tables: %v", err)
	}

	packet := binary.LittleEndian.AppendUint32(nil, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet,
		protocol.SVCPlayerInfo, 0x00, 0x00, 0x00,
		0x34, 0x12, 0x78, 0x56, 0xbc, 0x9a, 0xde,
		protocol.SVCNOP,
	)

	encodeState := state.NewPacket(0)
	plan, ok := qizmocodec.NewLinkEncoder(frequencies, encodeState).PlanPacket(packet, 0)
	if !ok {
		t.Fatal("test packet was not supported")
	}
	rangeEncoder := rangeenc.New()
	if err := plan.EncodeBody(rangeEncoder); err != nil {
		t.Fatalf("encode compressed body: %v", err)
	}
	linkPacket := append([]byte{1, 0, 1, 0x40}, rangeEncoder.Finish()...)

	decoder := newTestServerDecoder(t)
	got, err := decoder.Decode(linkPacket)
	if err != nil {
		t.Fatalf("decode link packet: %v", err)
	}
	if !bytes.Equal(got, packet) {
		t.Fatalf("decoded packet mismatch\ngot  %x\nwant %x", got, packet)
	}

	raw := binary.LittleEndian.AppendUint32(nil, protocol.QWConnectionlessSequence)
	raw = append(raw, []byte("c123456789")...)
	got, err = decoder.Decode(raw)
	if err != nil {
		t.Fatalf("decode raw packet: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("raw packet changed: got %x, want %x", got, raw)
	}
}

func TestServerDecoderPreservesReliableBits(t *testing.T) {
	frequencies, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("load frequency tables: %v", err)
	}
	packet := binary.LittleEndian.AppendUint32(nil, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet,
		protocol.SVCPlayerInfo, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0,
		protocol.SVCNOP,
	)
	plan, ok := qizmocodec.NewLinkEncoder(frequencies, state.NewPacket(0)).PlanPacket(packet, 0)
	if !ok {
		t.Fatal("test packet was not supported")
	}
	rangeEncoder := rangeenc.New()
	if err := plan.EncodeBody(rangeEncoder); err != nil {
		t.Fatalf("encode compressed body: %v", err)
	}
	linkPacket := append([]byte{1, 0x80, 1, 0xc0}, rangeEncoder.Finish()...)

	decoder := newTestServerDecoder(t)
	got, err := decoder.Decode(linkPacket)
	if err != nil {
		t.Fatalf("decode link packet: %v", err)
	}
	if sequence := binary.LittleEndian.Uint32(got); sequence != 0x80000001 {
		t.Fatalf("sequence = %#08x, want %#08x", sequence, uint32(0x80000001))
	}
	if ack := binary.LittleEndian.Uint32(got[protocol.QWPacketAckOffset:]); ack != 0x80000001 {
		t.Fatalf("ack = %#08x, want %#08x", ack, uint32(0x80000001))
	}
}

func TestServerDecoderUsesClientClockAndPreservesLinkEntityDeltas(t *testing.T) {
	frequencies, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("load frequency tables: %v", err)
	}

	encodeState := state.NewPacket(0)
	messageEncoder := qizmocodec.NewLinkEncoder(frequencies, encodeState)
	decoder := newTestServerDecoder(t)

	steps := []struct {
		name     string
		sequence uint32
		packet   []byte
		want     []byte
	}{
		{
			name:     "full snapshot",
			sequence: 1,
			packet:   testServerPacket(1, 100, []byte{0, 0}),
			want:     testServerPacket(1, 100, []byte{0, 0}),
		},
		{
			name:     "entity added",
			sequence: 2,
			packet:   testServerPacket(2, 120, []byte{0x21, 0, 0, 0}),
			want:     testLinkDeltaPacket(2, 120, 1, []byte{0x21, 0}),
		},
		{
			name:     "entity removed",
			sequence: 3,
			packet:   testServerPacket(3, 140, []byte{0, 0}),
			want:     testLinkDeltaPacket(3, 140, 2, []byte{0x21, 0x40}),
		},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			decoder.ObserveClientPacket(testClientMovePacket(step.sequence))
			encodeState.SetCommandScale(step.sequence, testCommandMsec)

			plan, ok := messageEncoder.PlanPacket(step.packet, step.sequence-1)
			if !ok {
				t.Fatal("packet was not supported")
			}
			rangeEncoder := rangeenc.New()
			if err := plan.EncodeBody(rangeEncoder); err != nil {
				t.Fatalf("encode packet: %v", err)
			}
			linkPacket := []byte{
				byte(step.sequence), byte(step.sequence >> 8),
				byte(step.sequence), byte(step.sequence>>8) | 0x40,
			}
			linkPacket = append(linkPacket, rangeEncoder.Finish()...)

			got, err := decoder.Decode(linkPacket)
			if err != nil {
				t.Fatalf("decode packet: %v", err)
			}
			if !bytes.Equal(got, step.want) {
				t.Fatalf("packet mismatch\ngot  %x\nwant %x", got, step.want)
			}
			plan.Commit()
		})
	}
}

func testClientMovePacket(sequence uint32) []byte {
	packet := binary.LittleEndian.AppendUint32(nil, sequence)
	packet = binary.LittleEndian.AppendUint32(packet, sequence)
	packet = append(packet, 0, 0)
	packet = append(packet,
		protocol.CLCMove, 0, 0,
		0, testCommandMsec,
		0, testCommandMsec,
		0, testCommandMsec,
	)
	return packet
}

func testServerPacket(sequence uint32, originX uint16, entities []byte) []byte {
	packet := binary.LittleEndian.AppendUint32(nil, sequence)
	packet = binary.LittleEndian.AppendUint32(packet, sequence)
	packet = append(packet, protocol.SVCPlayerInfo, 0, 0, 0)
	packet = binary.LittleEndian.AppendUint16(packet, originX)
	packet = append(packet, 0, 0, 0, 0, 1)
	packet = append(packet, protocol.SVCPacketEntities)
	packet = append(packet, entities...)
	return append(packet, protocol.SVCNOP)
}

func testLinkDeltaPacket(
	sequence uint32,
	originX uint16,
	base byte,
	entityDeltas []byte,
) []byte {
	packet := binary.LittleEndian.AppendUint32(nil, sequence)
	packet = binary.LittleEndian.AppendUint32(packet, sequence)
	packet = append(packet, protocol.SVCPlayerInfo, 0, 0, 0)
	packet = binary.LittleEndian.AppendUint16(packet, originX)
	packet = append(packet, 0, 0, 0, 0, 1)
	packet = append(packet, protocol.SVCDeltaPacketEntities, base)
	packet = append(packet, entityDeltas...)
	packet = append(packet, 0, 0, protocol.SVCNOP)
	return packet
}
