package rangedec

import (
	"errors"
	"io"
	"testing"
)

func TestStrictDecodeDoesNotPadInput(t *testing.T) {
	cumulative := []uint32{0x7fffffff, 0xffffffff}
	decoder, err := New([]byte{0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if _, err := decoder.DecodeSymbolStrict(cumulative, 2); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := decoder.DecodeSymbolStrict(cumulative, 2); !errors.Is(err, io.EOF) {
		t.Fatalf("strict decode error = %v, want EOF", err)
	}
	if _, err := decoder.DecodeSymbol(cumulative, 2); err != nil {
		t.Fatalf("padded decode: %v", err)
	}
}
