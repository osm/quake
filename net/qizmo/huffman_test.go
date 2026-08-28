package qizmo

import (
	"testing"

	"github.com/osm/quake/qizmo/freq"
)

func TestHuffmanCodesMatchQizmo(t *testing.T) {
	tests := []struct {
		model uint32
		want  []uint32
	}{
		{
			model: freq.CLCSequenceDelta,
			want: []uint32{
				0xc9578014, 0x80000002, 0x00000001, 0xe0000003,
				0xc0000005, 0xd0000004, 0xcc000006, 0xca000007,
			},
		},
		{
			model: freq.CLCType,
			want: []uint32{
				0xc0000002, 0x9f20000c, 0x9f30000c, 0x00000001,
				0xa0000003, 0x9e000008, 0x9f40000c, 0x9f50000c,
			},
		},
		{
			model: freq.CLCMoveChecksum,
			want: []uint32{
				0xa3000008, 0xde000008, 0x7f000008, 0x2e000008,
				0xf1000008, 0x57000008, 0x3b000008, 0x20000008,
			},
		},
		{
			model: freq.CLCStringByte,
			want: []uint32{
				0x00000003, 0x35da0010, 0x35db0010, 0x35dc0010,
				0x35dd0010, 0x35de0010, 0x35df0010, 0xafc00010,
			},
		},
	}

	for _, test := range tests {
		codes, err := buildHuffmanCodes(freq.DefaultCompressDat, test.model)
		if err != nil {
			t.Fatalf("build model %#x: %v", test.model, err)
		}
		for symbol, want := range test.want {
			if got := codes[symbol]; got != want {
				t.Errorf("model %#x symbol %d = %#08x, want %#08x", test.model, symbol, got, want)
			}
		}
	}
}

func TestBitWriterUsesQizmoBitOrder(t *testing.T) {
	writer := &bitWriter{}
	writer.writeCode(0x80000002) // 10
	writer.writeCode(0xe0000003) // 111
	writer.writeByte(0x5a)

	want := []byte{0xba, 0xd0}
	if len(writer.data) != len(want) {
		t.Fatalf("encoded size = %d, want %d", len(writer.data), len(want))
	}
	for i := range want {
		if writer.data[i] != want[i] {
			t.Fatalf("encoded bits = %x, want %x", writer.data, want)
		}
	}
}
