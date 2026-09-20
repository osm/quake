package codec

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/osm/quake/demo/mvz/frequency"
)

const compatibilityDir = "../../testdata/compatibility"

type compatibilityVector struct {
	name    string
	version uint16
	profile byte
}

var compatibilityRangeVectors = []compatibilityVector{
	{name: "version-1-profile-1", version: FormatVersion, profile: frequency.Profile1},
	{name: "version-1-profile-2", version: FormatVersion, profile: frequency.Profile2},
	{name: "version-1-profile-3", version: FormatVersion, profile: frequency.Profile3},
	{name: "version-1-profile-4", version: FormatVersion, profile: frequency.Profile4},
}

func readCompatibilityMVD(t testing.TB) []byte {
	t.Helper()
	path := filepath.Join(compatibilityDir, "version-1.mvd")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read compatibility MVD: %v", err)
	}
	return input
}

func TestCompatibilityVectors(t *testing.T) {
	input := readCompatibilityMVD(t)
	updateVectors := os.Getenv("MVZ_UPDATE_GOLDENS") != ""
	for _, vector := range compatibilityRangeVectors {
		t.Run(vector.name, func(t *testing.T) {
			resources := compatibilityResources(t, vector.version, vector.profile)
			encoded, err := encodeWithWorkers(input, resources, vector.version, 1)
			if err != nil {
				t.Fatalf("encode compatibility vector: %v", err)
			}
			checkCompatibilityVector(
				t, vector.name, encoded, input, updateVectors,
			)
		})
	}
}

func checkCompatibilityVector(
	t *testing.T,
	name string,
	encoded []byte,
	input []byte,
	update bool,
) {
	t.Helper()
	path := filepath.Join(compatibilityDir, name+".mvz")
	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create compatibility directory: %v", err)
		}
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			t.Fatalf("write compatibility vector: %v", err)
		}
		return
	}
	golden, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read compatibility vector: %v", err)
	}
	if !bytes.Equal(encoded, golden) {
		t.Fatalf(
			"encoded bytes changed: got %d bytes, want %d",
			len(encoded),
			len(golden),
		)
	}
	decoded, err := Decode(golden)
	if err != nil {
		t.Fatalf("decode compatibility vector: %v", err)
	}
	if !bytes.Equal(decoded, input) {
		t.Fatal("decoded data differs from the original demo")
	}
}

func compatibilityResources(
	t *testing.T,
	version uint16,
	profile byte,
) *RangeResources {
	t.Helper()
	resources, err := resourcesForVersion(version)
	if err != nil {
		t.Fatalf("load MVZ version %d: %v", version, err)
	}
	selected, ok := findProfile(resources.profiles, profile)
	if !ok {
		t.Fatalf("MVZ version %d has no profile %d", version, profile)
	}
	forced, err := newResources(selected)
	if err != nil {
		t.Fatalf("force MVZ version %d profile %d: %v", version, profile, err)
	}
	return forced
}

func readCompatibilityVector(t testing.TB, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(compatibilityDir, name+".mvz"))
	if err != nil {
		t.Fatalf("read compatibility vector: %v", err)
	}
	return data
}
