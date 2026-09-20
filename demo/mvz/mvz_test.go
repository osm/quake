package mvz_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/osm/quake/demo/mvz"
	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/tooling"
	"github.com/osm/quake/protocol"
)

func TestRoundTrip(t *testing.T) {
	// These checksums pin the encoded MVZ bytes, including chunk and profile choices.
	tests := []struct {
		filePath string
		checksum string
	}{
		{
			filePath: "../mvd/testdata/demo1.mvd",
			checksum: "af67e52fab46478e831a06798b4669ee154be24e960104ed56fbecbbdbef12d5",
		},
		{
			filePath: "../mvd/testdata/demo2.mvd",
			checksum: "a972999e2ee2f4e836ec0d0ace1504de52fd0e7b8b92a2cd3186c0180a41e5d4",
		},
		{
			filePath: "../mvd/testdata/demo3.mvd",
			checksum: "f4f799ec689c51bbde4c3c5a18f6a792bbcaab7589017116ac3abe7a80204ce8",
		},
		{
			filePath: "../mvd/testdata/demo4.mvd",
			checksum: "ea29bc761cda5a9ba23eabe57987cf2ac2433add016cd3b2a77eb548fdfb961e",
		},
	}

	for _, test := range tests {
		t.Run(filepath.Base(test.filePath), func(t *testing.T) {
			input, err := os.ReadFile(test.filePath)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			encoded, err := mvz.Encode(input)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if got := binary.LittleEndian.Uint16(encoded[4:6]); got != mvz.FormatVersion {
				t.Fatalf("version %d, want %d", got, mvz.FormatVersion)
			}
			checksum := fmt.Sprintf("%x", sha256.Sum256(encoded))
			if checksum != test.checksum {
				t.Fatalf("encoded checksum %s, want %s", checksum, test.checksum)
			}
			checkDemoRoundTrip(t, encoded, input)
			if len(encoded) >= len(input) {
				t.Fatalf(
					"fixture did not compress: got %d bytes, input %d",
					len(encoded),
					len(input),
				)
			}
		})
	}
	t.Run("invalid workers", func(t *testing.T) {
		empty, err := mvz.Encode(nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, workers := range []int{0, -1} {
			encoded, err := mvz.EncodeWithWorkers(nil, workers)
			if err == nil || encoded != nil {
				t.Fatalf("encode with %d workers: got (%v, %v)", workers, encoded, err)
			}
			decoded, err := mvz.DecodeWithWorkers(empty, workers)
			if err == nil || decoded != nil {
				t.Fatalf("decode with %d workers: got (%v, %v)", workers, decoded, err)
			}
		}
	})
	t.Run("empty", func(t *testing.T) {
		encoded, err := mvz.Encode(nil)
		if err != nil {
			t.Fatal(err)
		}
		checkDemoRoundTrip(t, encoded, nil)
	})
	t.Run("long player run", func(t *testing.T) {
		// A run longer than the wire byte allows must be split without loss.
		player := []byte{protocol.SVCPlayerInfo, 0, 0, 0, 0}
		body := bytes.Repeat(player, 256)
		input := binary.LittleEndian.AppendUint32([]byte{0, 6}, uint32(len(body)))
		input = append(input, body...)
		encoded, err := mvz.Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		checkDemoRoundTrip(t, encoded, input)
	})
}

func TestTrainingRangeResources(t *testing.T) {
	input, err := os.ReadFile("testdata/compatibility/version-1.mvd")
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := frequency.SetV1()
	if err != nil {
		t.Fatal(err)
	}
	models := profiles[0].Models
	modelData, err := models.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal models: %v", err)
	}
	resources, err := tooling.LoadRangeResources(modelData)
	if err != nil {
		t.Fatalf("load resources: %v", err)
	}
	encoded, err := tooling.EncodeWithResources(input, resources)
	if err != nil {
		t.Fatalf("training encode: %v", err)
	}
	version := binary.LittleEndian.Uint16(encoded[4:6])
	if version != tooling.TrainingVersion {
		t.Fatalf("version %#x, want training %#x", version, tooling.TrainingVersion)
	}
	if _, err := mvz.Decode(encoded); err == nil {
		t.Fatal("ordinary mvz.Decode accepted training output")
	}
	decoded, err := tooling.DecodeWithResources(encoded, resources)
	if err != nil {
		t.Fatalf("training decode: %v", err)
	}
	if !bytes.Equal(decoded, input) {
		t.Fatal("training decoded data differs")
	}
	baseline, err := mvz.Encode(input)
	if err != nil {
		t.Fatalf("baseline encode: %v", err)
	}
	badMagic := bytes.Clone(encoded)
	badMagic[0] ^= 1
	for _, data := range [][]byte{baseline, nil, encoded[:5], badMagic} {
		decoded, err := tooling.DecodeWithResources(data, resources)
		if err == nil || decoded != nil {
			t.Fatal("training decoder accepted invalid input or returned partial output")
		}
	}
}

func checkDemoRoundTrip(t *testing.T, encoded, want []byte) {
	t.Helper()
	decoded, err := mvz.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !bytes.Equal(decoded, want) {
		t.Fatal("decoded data differs from the original demo")
	}
	for _, workers := range []int{1, 2, 8} {
		parallel, err := mvz.EncodeWithWorkers(want, workers)
		if err != nil || !bytes.Equal(parallel, encoded) {
			t.Fatalf("%d encoding workers: bytes differ: %v", workers, err)
		}
		decoded, err := mvz.DecodeWithWorkers(encoded, workers)
		if err != nil || !bytes.Equal(decoded, want) {
			t.Fatalf("%d decoding workers: bytes differ: %v", workers, err)
		}
	}
}
