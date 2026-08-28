package rangeenc_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/rangeenc"
)

func TestEncoderRoundTrip(t *testing.T) {
	cumulative := uniformCumulative(256)
	rng := rand.New(rand.NewSource(0x517a0))

	for _, n := range []int{1, 2, 3, 4, 31, 256, 4096} {
		want := make([]byte, n)
		if _, err := rng.Read(want); err != nil {
			t.Fatalf("random symbols: %v", err)
		}

		enc := rangeenc.New()
		for i, symbol := range want {
			if err := enc.EncodeSymbol(cumulative, uint32(symbol)); err != nil {
				t.Fatalf("encode symbol %d: %v", i, err)
			}
		}
		dec := rangedec.NewPadded(enc.Finish())
		got := make([]byte, n)
		for i := range got {
			symbol, err := dec.DecodeSymbol(cumulative, 0x100)
			if err != nil {
				t.Fatalf("decode symbol %d: %v", i, err)
			}
			got[i] = byte(symbol)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("round trip mismatch for %d symbols", n)
		}
	}
}

func TestEncoderPairedStringModelRoundTrip(t *testing.T) {
	combined := uniformCumulative(512)
	var cumulative0, cumulative1 [256]uint32
	copy(cumulative0[:], combined[:256])
	copy(cumulative1[:], combined[256:])
	want := []uint32{0, 0xff, 0x100, 0x1ff, 0x101, 1, 0x1fe}

	enc := rangeenc.New()
	for i, symbol := range want {
		if err := enc.EncodeSymbol2x256(&cumulative0, &cumulative1, symbol); err != nil {
			t.Fatalf("encode symbol %d: %v", i, err)
		}
	}
	dec := rangedec.NewPadded(enc.Finish())
	for i, expected := range want {
		got, err := dec.DecodeSymbolStrict2x256(&cumulative0, &cumulative1)
		if err != nil {
			t.Fatalf("decode symbol %d: %v", i, err)
		}
		if got != expected {
			t.Fatalf("symbol %d = %#x, want %#x", i, got, expected)
		}
	}
}

func TestEncoderRejectsEmptyInterval(t *testing.T) {
	enc := rangeenc.New()
	if err := enc.EncodeSymbol([]uint32{1, 1, 0xffffffff}, 1); err == nil {
		t.Fatal("expected empty interval error")
	}
}

func uniformCumulative(symbols int) []uint32 {
	cumulative := make([]uint32, symbols)
	for i := range cumulative {
		cumulative[i] = uint32((uint64(i+1) * 0xffffffff) / uint64(symbols))
	}
	return cumulative
}
