package qizmo

import (
	"errors"
	"testing"

	"github.com/osm/quake/protocol"
)

func TestCompressedLinkPacketHeader(t *testing.T) {
	var sequences LinkSequences
	sequences.Observe(0x00003ffe, 0x80003ffd)

	header, body, err := sequences.DecodeHeader([]byte{
		0x01, 0x00,
		0x00, 0xc0,
		0xaa, 0xbb,
	})
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}
	if header.Sequence != 0x00004001 {
		t.Fatalf("sequence = %#08x, want %#08x", header.Sequence, uint32(0x00004001))
	}
	if header.Ack != 0x80004000 {
		t.Fatalf("ack = %#08x, want %#08x", header.Ack, uint32(0x80004000))
	}
	if len(body) != 2 || body[0] != 0xaa || body[1] != 0xbb {
		t.Fatalf("body = %x, want aabb", body)
	}
}

func TestCompressedLinkPacketDetection(t *testing.T) {
	connectionless := []byte{0xff, 0xff, 0xff, 0xff, 'c'}
	if IsCompressedLinkPacket(connectionless) {
		t.Fatal("connectionless packet detected as compressed")
	}

	var sequences LinkSequences
	if _, _, err := sequences.DecodeHeader([]byte{1, 0, 2, 0}); !errors.Is(err, ErrNotCompressed) {
		t.Fatalf("DecodeHeader error = %v, want ErrNotCompressed", err)
	}
}

func TestEncodeHeaderRoundTrip(t *testing.T) {
	want := LinkHeader{Sequence: 0x80004002, Ack: 0x80004001, CLCDelta: true}
	wire := EncodeHeader(want)
	if wire != [LinkHeaderSize]byte{0x02, 0xc0, 0x01, 0xc0} {
		t.Fatalf("encoded header = %x", wire)
	}

	var sequences LinkSequences
	sequences.Observe(0x3fff, 0x3fff)
	got, body, err := sequences.DecodeHeader(wire[:])
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}
	if got != want {
		t.Fatalf("decoded header = %+v, want %+v", got, want)
	}
	if len(body) != 0 {
		t.Fatalf("decoded body = %x, want empty", body)
	}
}

func TestDecodeHeaderAcrossSequenceWrap(t *testing.T) {
	var sequences LinkSequences
	sequences.Observe(protocol.QWSequenceMask-1, protocol.QWSequenceMask-2)

	wire := EncodeHeader(LinkHeader{Sequence: 1, Ack: 0})
	got, _, err := sequences.DecodeHeader(wire[:])
	if err != nil {
		t.Fatalf("DecodeHeader: %v", err)
	}
	if got.Sequence != 1 || got.Ack != 0 {
		t.Fatalf("decoded header = %+v, want sequence 1 and ack 0", got)
	}
}
