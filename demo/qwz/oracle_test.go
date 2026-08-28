package qwz_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/osm/quake/demo/qwz"
	"github.com/osm/quake/demo/qwz/internal/oracle"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
)

func TestQizmoV291Oracle(t *testing.T) {
	seed, err := os.ReadFile("testdata/demo26.qwz")
	if err != nil {
		t.Fatalf("read seed: %v", err)
	}
	scenarios, err := oracle.Build(seed)
	if err != nil {
		t.Fatalf("build scenarios: %v", err)
	}
	manifestData, err := os.ReadFile("internal/oracle/testdata/qizmo-v291.json")
	if err != nil {
		t.Fatalf("read oracle manifest: %v", err)
	}
	manifest, err := oracle.ParseManifest(manifestData)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Results) != len(scenarios) {
		t.Fatalf("result count = %d, want %d scenarios", len(manifest.Results), len(scenarios))
	}
	resultsByName := make(map[string]oracle.Result, len(manifest.Results))
	for _, result := range manifest.Results {
		resultsByName[result.Name] = result
	}

	ft, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("frequency tables: %v", err)
	}
	packetAssets := assets.Embedded()
	for index, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			want, ok := resultsByName[scenario.Name]
			if !ok {
				t.Fatal("missing oracle result")
			}
			if want.Family != scenario.Family {
				t.Fatalf("family = %s, want %s", want.Family, scenario.Family)
			}
			if len(scenario.QWD) != want.InputSize || oracle.SHA256(scenario.QWD) != want.InputSHA256 {
				t.Fatalf("generated input no longer matches the oracle input")
			}

			encoded, err := qwz.Encode(scenario.QWD, ft)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if len(encoded) != want.QWZSize || oracle.SHA256(encoded) != want.QWZSHA256 {
				t.Fatalf(
					"QWZ mismatch: got %d bytes %s, want %d bytes %s",
					len(encoded), oracle.SHA256(encoded), want.QWZSize, want.QWZSHA256,
				)
			}
			decoded, err := qwz.Decode(encoded, ft, packetAssets)
			if err != nil {
				t.Fatalf("decode QWZ: %v", err)
			}
			if len(decoded) != want.DecodedSize || oracle.SHA256(decoded) != want.DecodedSHA256 {
				t.Fatalf(
					"decoded QWD mismatch: got %d bytes %s, want %d bytes %s",
					len(decoded), oracle.SHA256(decoded), want.DecodedSize, want.DecodedSHA256,
				)
			}
		})
		if index != 0 && index%500 == 0 {
			runtime.GC()
		}
	}
}
