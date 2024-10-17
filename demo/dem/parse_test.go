package dem

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/osm/quake/common/context"
)

type demTest struct {
	filePath string
	checksum string
}

var demTests = []demTest{
	{
		filePath: "testdata/demo1.dem",
		checksum: "893e5279e3a84fed0416a1101dfc2b6bca1ebd28787323daa8e22cefe54883c1",
	},
	{
		filePath: "testdata/demo2.dem",
		checksum: "150e6fc54c24a70a44012ba71473a5cbbec081e19204a9eae98249621fef00d7",
	},
	{
		filePath: "testdata/demo3.dem",
		checksum: "7bee6edc47fe563cbf8f1e873d524d1e4cef7d919b523606e200c0d7c269bd83",
	},
}

func TestParse(t *testing.T) {
	for _, dt := range demTests {
		t.Run(dt.filePath, func(t *testing.T) {
			data, err := ioutil.ReadFile(dt.filePath)
			if err != nil {
				t.Errorf("unable to open demo file, %v", err)
			}

			demo, err := Parse(context.New(), data)
			if err != nil {
				t.Errorf("unable to parse demo, %v", err)
			}

			h := sha256.New()
			h.Write(demo.Bytes())
			checksum := fmt.Sprintf("%x", h.Sum(nil))
			if checksum != dt.checksum {
				t.Errorf("sha256 checksums didn't match")
				t.Logf("output: %#v", checksum)
				t.Logf("expected: %#v", dt.checksum)
			}
		})
	}
}
