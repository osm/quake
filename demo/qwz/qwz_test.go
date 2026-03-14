package qwz_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/osm/quake/demo/qwz"
	"github.com/osm/quake/demo/qwz/assets"
	"github.com/osm/quake/demo/qwz/freq"
)

func TestDecodeFixtures(t *testing.T) {
	ft, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("new freq tables: %v", err)
	}

	da := assets.Assets{
		PrecacheModels:     assets.PrecacheModels,
		PrecacheSounds:     assets.PrecacheSounds,
		CenterPrintStrings: assets.EmbeddedStringTable(assets.CenterPrintStrings),
		PrintMode3Strings:  assets.EmbeddedStringTable(assets.PrintMode3Strings),
		PrintStrings:       assets.EmbeddedStringTable(assets.PrintStrings),
		SetInfoStrings:     assets.EmbeddedStringTable(assets.SetInfoStrings),
		StuffTextStrings:   assets.EmbeddedStringTable(assets.StuffTextStrings),
	}

	qwzPaths, err := filepath.Glob(filepath.Join("testdata", "*.qwz"))
	if err != nil {
		t.Fatalf("glob testdata demos: %v", err)
	}

	sort.Strings(qwzPaths)

	if len(qwzPaths) == 0 {
		t.Fatal("no .qwz fixtures found in testdata")
	}

	for _, qwzPath := range qwzPaths {
		name := strings.TrimSuffix(filepath.Base(qwzPath), ".qwz")
		qwdPath := filepath.Join("testdata", name+".qwd")

		if _, err := os.Stat(qwdPath); err != nil {
			t.Fatalf("missing fixture pair for %s: %v", name, err)
		}

		t.Run(name, func(t *testing.T) {
			qwzData, err := os.ReadFile(qwzPath)
			if err != nil {
				t.Fatalf("read %s: %v", qwzPath, err)
			}

			want, err := os.ReadFile(qwdPath)
			if err != nil {
				t.Fatalf("read %s: %v", qwdPath, err)
			}

			got, err := qwz.Decode(qwzData, ft, da)
			if err != nil {
				t.Fatalf("decode %s: %v", qwzPath, err)
			}

			if bytes.Equal(got, want) {
				return
			}

			diffAt := firstDiff(got, want)
			t.Fatalf(
				"decoded bytes mismatch at %d (got=%d want=%d)",
				diffAt,
				len(got),
				len(want),
			)
		})
	}
}

func firstDiff(got, want []byte) int {
	limit := len(got)
	if len(want) < limit {
		limit = len(want)
	}

	for i := 0; i < limit; i++ {
		if got[i] != want[i] {
			return i
		}
	}

	return limit
}
