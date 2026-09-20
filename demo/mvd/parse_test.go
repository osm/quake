package mvd

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/osm/quake/common/context"
)

type mvdTest struct {
	filePath string
	checksum string
}

var mvdTests = []mvdTest{
	{
		filePath: "testdata/demo1.mvd",
		checksum: "9489430c9513aed5b8e444df1b0d2040acb55de565c1b3e237dce51c8c103c57",
	},
	{
		filePath: "testdata/demo2.mvd",
		checksum: "d40c3864dd3c052a8379e7e589cd045814ec462f6cf4432a2c94ec740843e30d",
	},
	{
		filePath: "testdata/demo3.mvd",
		checksum: "3dd728aa90fdae100ade62d9b93220b7689f6ebdca6c986d3c2b1208b4e1e33c",
	},
	{
		filePath: "testdata/demo4.mvd",
		checksum: "1f7b2c0ad77608f431028c8e68904400fe406150875a51f793f3395bf3784c90",
	},
}

func TestParse(t *testing.T) {
	for _, mt := range mvdTests {
		t.Run(mt.filePath, func(t *testing.T) {
			data, err := os.ReadFile(mt.filePath)
			if err != nil {
				t.Fatalf("unable to open demo file, %v", err)
			}

			demo, err := Parse(context.New(), data)
			if err != nil {
				t.Fatalf("unable to parse demo, %v", err)
			}

			offset := 0
			for i := range demo.Data {
				raw := demo.Data[i].OriginalBytes()
				if len(raw) == 0 || len(raw) > len(data)-offset ||
					!bytes.Equal(raw, data[offset:offset+len(raw)]) {
					t.Fatalf("record %d does not retain its original bytes", i)
				}
				offset += len(raw)
			}
			if !bytes.Equal(demo.TrailingData, data[offset:]) {
				t.Fatal("trailing data does not retain its original bytes")
			}

			h := sha256.New()
			h.Write(demo.Bytes())
			checksum := fmt.Sprintf("%x", h.Sum(nil))
			if checksum != mt.checksum {
				t.Errorf("sha256 checksums didn't match")
				t.Logf("output: %#v", checksum)
				t.Logf("expected: %#v", mt.checksum)
			}
		})
	}
}

func TestConstructedDataHasNoOriginalBytes(t *testing.T) {
	if raw := (&Data{}).OriginalBytes(); raw != nil {
		t.Fatalf("constructed data has %d original bytes", len(raw))
	}
}

func TestParseRecordsPropagatesVisitorError(t *testing.T) {
	data, err := os.ReadFile("testdata/demo4.mvd")
	if err != nil {
		t.Fatalf("read demo: %v", err)
	}
	want := errors.New("stop")
	visits := 0
	_, err = ParseRecords(context.New(), data, func(*Data) error {
		visits++
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("parse records error %v, want %v", err, want)
	}
	if visits != 1 {
		t.Fatalf("visited %d records, want 1", visits)
	}
}

func TestParseRecordsRejectsNilVisitor(t *testing.T) {
	if _, err := ParseRecords(context.New(), nil, nil); err == nil {
		t.Fatal("ParseRecords accepted a nil visitor")
	}
}
