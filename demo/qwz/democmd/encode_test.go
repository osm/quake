package democmd_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/osm/quake/demo/qwd"
	"github.com/osm/quake/demo/qwz/democmd"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/rangeenc"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	ft := testFrequencyTables(t)

	random := rand.New(rand.NewSource(0xc0de))
	want := make([][democmd.PayloadSize]byte, 512)
	for i := range want {
		angles := [3]int16{int16(random.Uint32()), int16(random.Uint32()), int16(random.Uint32())}
		impulse := byte(random.Uint32())
		if impulse == 0 {
			impulse = 1
		}
		want[i] = commandPayload(
			byte(random.Uint32()),
			angles,
			int16(random.Uint32()),
			int16(random.Uint32()),
			int16(random.Uint32()),
			byte(random.Uint32()),
			impulse,
		)
	}

	enc := rangeenc.New()
	encodeState := democmd.NewState()
	for i, payload := range want {
		if err := democmd.Encode(enc, ft, encodeState, payload); err != nil {
			t.Fatalf("encode command %d: %v", i, err)
		}
	}
	dec := rangedec.NewPadded(enc.Finish())
	decodeState := democmd.NewState()
	for i, expected := range want {
		got, err := democmd.Decode(dec, ft, decodeState)
		if err != nil {
			t.Fatalf("decode command %d: %v", i, err)
		}
		if !bytes.Equal(got[:], expected[:]) {
			t.Fatalf("command %d mismatch\ngot  %x\nwant %x", i, got, expected)
		}
	}
}

func TestEncodeRejectsNonCanonicalCommand(t *testing.T) {
	ft := testFrequencyTables(t)
	payload := commandPayload(12, [3]int16{}, 0, 0, 0, 0, 8)
	payload[1] = 1
	if err := democmd.Encode(rangeenc.New(), ft, democmd.NewState(), payload); err == nil {
		t.Fatal("expected alignment padding error")
	}
}

func TestEncodeEmitsUnchangedNonzeroImpulse(t *testing.T) {
	ft := testFrequencyTables(t)

	enc := rangeenc.New()
	payload := commandPayload(0, [3]int16{}, 0, 0, 0, 0, 8)
	if err := democmd.Encode(enc, ft, democmd.NewState(), payload); err != nil {
		t.Fatalf("encode: %v", err)
	}
	dec := rangedec.NewPadded(enc.Finish())
	lo, err := dec.DecodeSymbol(ft.CumulativeRow(freq.CmdMaskLo), freq.Symbols)
	if err != nil {
		t.Fatalf("decode low mask: %v", err)
	}
	hi, err := dec.DecodeSymbol(ft.CumulativeRow(freq.CmdMaskHi), freq.Symbols)
	if err != nil {
		t.Fatalf("decode high mask: %v", err)
	}
	mask := uint16(lo) | uint16(hi)<<8
	if mask != qizmoprotocol.CMImpulse {
		t.Fatalf("encoded mask = %#04x, want %#04x", mask, uint16(qizmoprotocol.CMImpulse))
	}
}

func TestEncodeRejectsClearingRetainedImpulse(t *testing.T) {
	ft := testFrequencyTables(t)
	payload := commandPayload(0, [3]int16{}, 0, 0, 0, 0, 0)
	if err := democmd.Encode(rangeenc.New(), ft, democmd.NewState(), payload); err == nil {
		t.Fatal("expected retained impulse error")
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

func commandPayload(
	msec byte,
	angles [3]int16,
	forward, side, up int16,
	buttons, impulse byte,
) [democmd.PayloadSize]byte {
	command := qwd.Cmd{
		Msec:    msec,
		Forward: uint16(forward),
		Side:    uint16(side),
		Up:      uint16(up),
		Buttons: buttons,
		Impulse: impulse,
	}
	for i, angle := range angles {
		value := float32(float64(angle) * (360.0 / 65536.0))
		command.UserAngle[i] = value
		command.Angle[i] = value
	}

	var payload [democmd.PayloadSize]byte
	copy(payload[:], command.Bytes())
	return payload
}
