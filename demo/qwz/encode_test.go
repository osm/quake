package qwz_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/osm/quake/demo/qwd"
	"github.com/osm/quake/demo/qwz"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangedec"
)

var testDemoCommandCumulative = []uint32{
	0x80000000,
	0xfffffc00,
	0xfffffe00,
	0xffffffff,
}

func TestEncodeUsesRawModeForConnectionlessPacket(t *testing.T) {
	ft := testFrequencyTables(t)
	packet := binary.LittleEndian.AppendUint32(nil, protocol.QWConnectionlessSequence)
	packet = binary.LittleEndian.AppendUint32(packet, 0)
	packet = append(packet, 0xde, 0xad, 0xbe, 0xef)
	qwd := qwdReadRecord(packet)

	encoded, err := qwz.Encode(qwd, ft)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if mode := encodedReadMode(t, ft, encoded); mode != 0 {
		t.Fatalf("mode = %d, want raw mode 0", mode)
	}

	got, err := qwz.Decode(encoded, ft, assets.Assets{})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(got, qwd) {
		t.Fatalf("round trip mismatch\ngot  %x\nwant %x", got, qwd)
	}
}

func TestEncodeRejectsZeroLengthRead(t *testing.T) {
	if _, err := qwz.Encode(qwdReadRecord(nil), testFrequencyTables(t)); err == nil {
		t.Fatal("zero-length DEMO_READ was accepted")
	}
}

func TestNilFrequencyTables(t *testing.T) {
	if _, err := qwz.Encode(nil, nil); err == nil {
		t.Fatal("Encode accepted nil frequency tables")
	}
	if _, err := qwz.Decode(nil, nil, assets.Assets{}); err == nil {
		t.Fatal("Decode accepted nil frequency tables")
	}
}

func qwdReadRecord(packet []byte) []byte {
	recordSize := qwd.RecordHeaderSize + qwd.ReadSizeFieldSize + len(packet)
	record := make([]byte, qwd.RecordHeaderSize, recordSize)
	record[qwd.TimestampSize] = protocol.DemoRead
	record = binary.LittleEndian.AppendUint32(record, uint32(len(packet)))
	return append(record, packet...)
}

func encodedReadMode(t *testing.T, ft *freq.Tables, encoded []byte) uint32 {
	t.Helper()
	dec := rangedec.NewPadded(encoded)
	cmd, err := dec.DecodeSymbol(testDemoCommandCumulative, uint32(len(testDemoCommandCumulative)))
	if err != nil {
		t.Fatalf("decode outer command: %v", err)
	}
	if cmd != protocol.DemoRead {
		t.Fatalf("outer command = %d, want DEMO_READ", cmd)
	}
	mode, err := dec.DecodeSymbol(ft.CumulativeRow(freq.DemoMode), freq.Symbols)
	if err != nil {
		t.Fatalf("decode mode: %v", err)
	}
	return mode
}

func testFrequencyTables(t *testing.T) *freq.Tables {
	t.Helper()
	ft, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("new frequency tables: %v", err)
	}
	return ft
}
