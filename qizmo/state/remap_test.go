package state

import "testing"

func TestRemapIndexUsesFirstMatchingEntry(t *testing.T) {
	packet := NewPacket(0)
	packet.modelRemap[7] = 42
	packet.soundRemap[9] = 43

	if got, ok := packet.ModelRemapIndex(42); !ok || got != 7 {
		t.Fatalf("model remap index = (%d, %v), want (7, true)", got, ok)
	}
	if got, ok := packet.SoundRemapIndex(43); !ok || got != 9 {
		t.Fatalf("sound remap index = (%d, %v), want (9, true)", got, ok)
	}
}

func TestRemapIndexReportsMissingValue(t *testing.T) {
	packet := NewPacket(0)
	for i := range packet.modelRemap {
		packet.modelRemap[i] = 0
		packet.soundRemap[i] = 0
	}

	if _, ok := packet.ModelRemapIndex(1); ok {
		t.Fatal("missing model remap was reported as present")
	}
	if _, ok := packet.SoundRemapIndex(1); ok {
		t.Fatal("missing sound remap was reported as present")
	}
}
