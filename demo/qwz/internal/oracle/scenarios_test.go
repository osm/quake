package oracle

import (
	"os"
	"testing"
)

func TestScenarioCoverage(t *testing.T) {
	seed, err := os.ReadFile("../../testdata/demo26.qwz")
	if err != nil {
		t.Fatalf("read seed: %v", err)
	}
	scenarios, err := Build(seed)
	if err != nil {
		t.Fatalf("build scenarios: %v", err)
	}
	got := make(map[string]int)
	for _, scenario := range scenarios {
		got[scenario.Family]++
	}
	want := map[string]int{
		"baseline":        1,
		"usercmd":         318,
		"player_flags":    1088,
		"packet_entities": 2114,
		"services":        62,
		"history":         15,
		"demo_cmd":        45,
		"outer":           10,
	}
	if len(got) != len(want) {
		t.Fatalf("family count = %d, want %d: %#v", len(got), len(want), got)
	}
	for family, count := range want {
		if got[family] != count {
			t.Errorf("family %s count = %d, want %d", family, got[family], count)
		}
	}
	if len(scenarios) != 3653 {
		t.Fatalf("scenario count = %d, want 3653", len(scenarios))
	}
}
