package svc

import (
	"bytes"
	"testing"

	"github.com/osm/quake/common/context"
)

func TestConnectionlessPassthroughPreservesType(t *testing.T) {
	want := []byte{0xff, 0xff, 0xff, 0xff, 'r'}
	packet, err := Parse(context.New(), want)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := packet.Bytes(); !bytes.Equal(got, want) {
		t.Fatalf("round trip = %x, want %x", got, want)
	}
}
