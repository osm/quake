package qwd

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/osm/quake/common/context"
)

type qwdTest struct {
	filePath string
	checksum string
}

var qwdTests = []qwdTest{
	{
		filePath: "testdata/demo1.qwd",
		checksum: "849cf01cb3fe5c7161625dafdf03178536048d093eda885ca9b37fc673c3c72a",
	},
	{
		filePath: "testdata/demo2.qwd",
		checksum: "c90f0a6c9ba79fc7f84d90b12d923a3ad2c0e54849d8df6fd913264f147931b2",
	},
	{
		filePath: "testdata/demo3.qwd",
		checksum: "b816f366489d22ed70b84f105a19ce63810fee7295e3b3612fdf7e09c26fc9e9",
	},
}

func TestParse(t *testing.T) {
	for _, qt := range qwdTests {
		t.Run(qt.filePath, func(t *testing.T) {
			data, err := ioutil.ReadFile(qt.filePath)
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
			if checksum != qt.checksum {
				t.Errorf("sha256 checksums didn't match")
				t.Logf("output: %#v", checksum)
				t.Logf("expected: %#v", qt.checksum)
			}
		})
	}
}
