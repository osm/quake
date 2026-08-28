package qizmo_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	qizmonet "github.com/osm/quake/net/qizmo"
	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
)

func TestServerCodecRoundTripFullAndDeltaPackets(t *testing.T) {
	encoder, decoder := newTestServerCodec(t)

	first := testServerPacket(1, 100, []byte{0, 0})
	binary.LittleEndian.PutUint32(first[protocol.QWPacketAckOffset:], 0x54)
	second := testLinkDeltaPacket(2, 120, 1, nil)
	binary.LittleEndian.PutUint32(second, 0x80000002)
	binary.LittleEndian.PutUint32(second[protocol.QWPacketAckOffset:], 0x80000055)

	for i, packet := range [][]byte{first, second} {
		sequence := uint32(i + 1)
		clientPacket := testClientMovePacket(sequence)
		encoder.ObserveClientPacket(clientPacket)
		decoder.ObserveClientPacket(clientPacket)

		encoded, err := encoder.Encode(packet)
		if err != nil {
			t.Fatalf("encode packet %d: %v", sequence, err)
		}
		if !qizmoprotocol.IsCompressedLinkPacket(encoded) {
			t.Fatalf("packet %d remained raw (%d bytes)", sequence, len(encoded))
		}
		if len(encoded) >= len(packet) {
			t.Fatalf("packet %d grew from %d to %d bytes", sequence, len(packet), len(encoded))
		}

		got, err := decoder.Decode(encoded)
		if err != nil {
			t.Fatalf("decode packet %d: %v", sequence, err)
		}
		if !bytes.Equal(got, packet) {
			t.Fatalf("server codec packet %d mismatch\ngot  %x\nwant %x\nwire %x", sequence, got, packet, encoded)
		}
	}
}

func TestServerEncoderRawFallbackUpdatesHistory(t *testing.T) {
	encoder, decoder := newTestServerCodec(t)

	canonical := testServerPacket(1, 100, []byte{0, 0})
	raw := append([]byte(nil), canonical[:protocol.QWServerPacketHeaderSize]...)
	raw = append(raw, protocol.SVCNOP) // A leading svc_nop makes this packet shape unsupported.
	raw = append(raw, canonical[protocol.QWServerPacketHeaderSize:]...)
	binary.LittleEndian.PutUint32(raw, 0x80000001)
	encoder.ObserveClientPacket(testClientMovePacket(1))
	decoder.ObserveClientPacket(testClientMovePacket(1))
	encoded, err := encoder.Encode(raw)
	if err != nil {
		t.Fatalf("encode raw seed: %v", err)
	}
	if !bytes.Equal(encoded, raw) {
		t.Fatalf("small seed was compressed: %x", encoded)
	}
	if _, err := decoder.Decode(encoded); err != nil {
		t.Fatalf("decode raw seed: %v", err)
	}

	delta := testLinkDeltaPacket(2, 120, 1, nil)
	encoder.ObserveClientPacket(testClientMovePacket(2))
	decoder.ObserveClientPacket(testClientMovePacket(2))
	encoded, err = encoder.Encode(delta)
	if err != nil {
		t.Fatalf("encode delta: %v", err)
	}
	if !qizmoprotocol.IsCompressedLinkPacket(encoded) {
		t.Fatal("delta after raw seed remained raw")
	}
	got, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("decode delta: %v", err)
	}
	if !bytes.Equal(got, delta) {
		t.Fatalf("delta mismatch\ngot  %x\nwant %x", got, delta)
	}
}

func TestServerEncoderHonorsClientResyncRequest(t *testing.T) {
	state := qizmonet.NewEndpointState()
	clientDecoder, err := qizmonet.NewClientDecoder(state)
	if err != nil {
		t.Fatalf("NewClientDecoder: %v", err)
	}
	encoder, err := qizmonet.NewServerEncoder(state)
	if err != nil {
		t.Fatalf("NewServerEncoder: %v", err)
	}

	encoder.ObserveClientPacket(testClientMovePacket(1))
	first := testServerPacket(1, 100, []byte{0, 0})
	if encoded, err := encoder.Encode(first); err != nil {
		t.Fatalf("encode history seed: %v", err)
	} else if !qizmoprotocol.IsCompressedLinkPacket(encoded) {
		t.Fatal("history seed remained raw")
	}

	request := binary.LittleEndian.AppendUint32(nil, 1)
	request = binary.LittleEndian.AppendUint32(request, 1)
	request = binary.LittleEndian.AppendUint16(request, 1)
	request = append(request, qizmoprotocol.CLCRequestS2CResync)
	ordinary, extensions, err := clientDecoder.DecodeWithExtensions(request)
	if err != nil {
		t.Fatalf("decode resync request: %v", err)
	}
	if len(ordinary) != protocol.QWClientPacketHeaderSize || len(extensions) != 1 ||
		extensions[0].Opcode != qizmoprotocol.CLCRequestS2CResync {
		t.Fatalf("separated resync = packet %x, extensions %#v", ordinary, extensions)
	}
	encoder.ObserveClientPacket(testClientMovePacket(2))
	second := testServerPacket(2, 120, []byte{0, 0})
	encoded, err := encoder.Encode(second)
	if err != nil {
		t.Fatalf("encode resynchronized packet: %v", err)
	}
	if !qizmoprotocol.IsCompressedLinkPacket(encoded) {
		t.Fatal("resynchronized packet remained raw")
	}

	// A decoder with no S2C packet history can consume this packet. It still
	// observes the client clock because that prediction state is independent.
	freshDecoder := newTestServerDecoder(t)
	freshDecoder.ObserveClientPacket(testClientMovePacket(2))
	got, err := freshDecoder.Decode(encoded)
	if err != nil {
		t.Fatalf("fresh decoder rejected resynchronized packet: %v", err)
	}
	if !bytes.Equal(got, second) {
		t.Fatalf("resynchronized packet mismatch\ngot  %x\nwant %x", got, second)
	}

	encoder.ObserveClientPacket(testClientMovePacket(3))
	third := testLinkDeltaPacket(3, 140, 2, nil)
	encoded, err = encoder.Encode(third)
	if err != nil {
		t.Fatalf("encode packet after resynchronization: %v", err)
	}
	if !qizmoprotocol.IsCompressedLinkPacket(encoded) {
		t.Fatal("packet after resynchronization remained raw")
	}
	freshDecoder.ObserveClientPacket(testClientMovePacket(3))
	got, err = freshDecoder.Decode(encoded)
	if err != nil {
		t.Fatalf("decode packet after resynchronization: %v", err)
	}
	if !bytes.Equal(got, third) {
		t.Fatalf("packet after resynchronization mismatch\ngot  %x\nwant %x", got, third)
	}

	historylessDecoder := newTestServerDecoder(t)
	historylessDecoder.ObserveClientPacket(testClientMovePacket(3))
	got, err = historylessDecoder.Decode(encoded)
	if err != nil {
		t.Fatalf("decode packet without history: %v", err)
	}
	if got != nil {
		t.Fatal("resync request affected more than one server packet")
	}
}

func newTestServerCodec(t *testing.T) (*qizmonet.ServerEncoder, *qizmonet.ServerDecoder) {
	t.Helper()
	encoder, err := qizmonet.NewServerEncoder(qizmonet.NewEndpointState())
	if err != nil {
		t.Fatalf("NewServerEncoder: %v", err)
	}
	return encoder, newTestServerDecoder(t)
}

func newTestServerDecoder(t *testing.T) *qizmonet.ServerDecoder {
	t.Helper()
	decoder, err := qizmonet.NewServerDecoder(qizmonet.NewEndpointState())
	if err != nil {
		t.Fatalf("NewServerDecoder: %v", err)
	}
	return decoder
}
