package codec

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Worker scheduling must not change output or which input error is reported.
func FuzzEncodeMalformedInput(f *testing.F) {
	resources, err := resourcesForVersion(FormatVersion)
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte(nil))
	seed := readCompatibilityMVD(f)
	f.Add(seed)
	f.Add(bytes.Repeat(seed, 2*rangeChunkTargetSize/len(seed)+1))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 2<<20 {
			t.Skip("limit fuzz parsing and preparation memory")
		}
		parallel, err := encodeWithWorkers(
			input, resources, FormatVersion, 8,
		)
		serial, serialErr := encodeWithWorkers(
			input, resources, FormatVersion, 1,
		)
		if (err == nil) != (serialErr == nil) ||
			(err != nil && err.Error() != serialErr.Error()) ||
			!bytes.Equal(parallel, serial) {
			t.Fatalf("parallel and serial encoding differ: %v, %v", err, serialErr)
		}
		if err == nil {
			decoded, err := Decode(parallel)
			if err != nil || !bytes.Equal(decoded, input) {
				t.Fatalf("successful encode did not round trip: %v", err)
			}
		}
	})
}

// Malformed input must fail safely, with the same result regardless of worker count.
func FuzzDecodeMalformedInput(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte("MVZ\x1a\x01\x00\x00"))
	var multi bytes.Buffer
	if err := writeFileHeader(&multi, FormatVersion); err != nil {
		f.Fatal(err)
	}
	for profile := 1; profile <= 4; profile++ {
		name := "version-1-profile-" + strconv.Itoa(profile) + ".mvz"
		path := filepath.Join(compatibilityDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			f.Fatalf("read fuzz seed %s: %v", path, err)
		}
		f.Add(data)
		multi.Write(data[headerSize : len(data)-endEntrySize])
	}
	if err := writeEndEntry(&multi); err != nil {
		f.Fatal(err)
	}
	f.Add(multi.Bytes())

	f.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := DecodeWithWorkers(data, 4)
		if errors.Is(err, io.EOF) {
			t.Fatalf("Decode reported normal EOF instead of success or corruption: %v", err)
		}
		serial, serialErr := DecodeWithWorkers(data, 1)
		if (err == nil) != (serialErr == nil) ||
			(err != nil && err.Error() != serialErr.Error()) ||
			!bytes.Equal(decoded, serial) {
			t.Fatalf("parallel and serial decoding differ: %v, %v", err, serialErr)
		}
	})
}
