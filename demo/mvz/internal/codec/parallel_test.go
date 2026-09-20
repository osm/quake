package codec

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func multiProfileChunks(t *testing.T) ([]byte, []byte, []int) {
	t.Helper()
	var encoded bytes.Buffer
	if err := writeFileHeader(&encoded, FormatVersion); err != nil {
		t.Fatal(err)
	}
	var starts []int
	for _, vector := range compatibilityRangeVectors {
		data := readCompatibilityVector(t, vector.name)
		starts = append(starts, encoded.Len())
		encoded.Write(data[headerSize : len(data)-endEntrySize])
	}
	if err := writeEndEntry(&encoded); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes(), bytes.Repeat(readCompatibilityMVD(t), len(starts)), starts
}

func TestParallelDecodeMixedProfiles(t *testing.T) {
	encoded, want, _ := multiProfileChunks(t)
	for _, workers := range []int{1, 2, 4, 8} {
		decoded, err := DecodeWithWorkers(encoded, workers)
		if err != nil || !bytes.Equal(decoded, want) {
			t.Fatalf("%d workers: mixed-profile decode differs: %v", workers, err)
		}
	}
}

func TestParallelDecodeErrors(t *testing.T) {
	encoded, _, starts := multiProfileChunks(t)
	var cases [][]byte
	for _, start := range starts {
		badCRC := bytes.Clone(encoded)
		badCRC[start+5] ^= 1
		cases = append(cases, badCRC)
		// A later framing error must not hide an earlier checksum error.
		cases = append(cases, badCRC[:len(badCRC)-endEntrySize])
		cases = append(cases, append(bytes.Clone(badCRC), 0xff))
		badProfile := bytes.Clone(encoded)
		badProfile[start+chunkHeaderSize+4] = 0xff
		cases = append(cases, badProfile)
		badSize := bytes.Clone(encoded)
		binary.LittleEndian.PutUint32(badSize[start+1:], maxChunkSize+1)
		cases = append(cases, badSize)
		lengths := []int{
			start, start + 1, start + chunkHeaderSize, start + chunkHeaderSize + 1,
		}
		for _, length := range lengths {
			cases = append(cases, bytes.Clone(encoded[:length]))
		}
	}
	cases = append(cases, encoded[:len(encoded)-1], append(bytes.Clone(encoded), 0xff))
	for i, data := range cases {
		_, want := DecodeWithWorkers(data, 1)
		if want == nil {
			t.Fatalf("case %d is not malformed", i)
		}
		for _, workers := range []int{2, 8} {
			decoded, err := DecodeWithWorkers(data, workers)
			if err == nil || err.Error() != want.Error() || decoded != nil {
				t.Fatalf("case %d, %d workers: got %v, want %v", i, workers, err, want)
			}
		}
	}
}

func TestDecodeChunksAboveEncoderTarget(t *testing.T) {
	resources, err := resourcesForVersion(FormatVersion)
	if err != nil {
		t.Fatal(err)
	}
	var encoded, want bytes.Buffer
	if err := writeFileHeader(&encoded, FormatVersion); err != nil {
		t.Fatal(err)
	}
	profile := resources.profiles[0].id
	for i := byte(0); i < 3; i++ {
		record := rangeRecord{
			timestamp: i, command: 2, payload: bytes.Repeat([]byte{0x31}, 2<<20),
		}
		chunk := rangeChunkInput{records: []rangeRecord{record}, original: record.bytes()}
		err := encodeChunk(&encoded, resources, &profile, chunk, 1)
		if err != nil {
			t.Fatal(err)
		}
		want.Write(chunk.original)
	}
	if err := writeEndEntry(&encoded); err != nil {
		t.Fatal(err)
	}
	for _, workers := range []int{1, 8} {
		decoded, err := DecodeWithWorkers(encoded.Bytes(), workers)
		if err != nil || !bytes.Equal(decoded, want.Bytes()) {
			t.Fatalf(
				"%d workers: rejected valid chunks above the encoder target: %v",
				workers, err,
			)
		}
	}
}

func TestParallelEncodingMatchesSerial(t *testing.T) {
	path := filepath.Join("..", "..", "..", "mvd", "testdata", "demo1.mvd")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	resources, err := resourcesForVersion(FormatVersion)
	if err != nil {
		t.Fatalf("load version 1 resources: %v", err)
	}
	serial, err := encodeWithWorkers(
		input,
		resources,
		FormatVersion,
		1,
	)
	if err != nil {
		t.Fatalf("serial encode: %v", err)
	}
	chunks, err := InspectChunks(serial)
	if err != nil {
		t.Fatal(err)
	}
	// This fixture's records fit into the minimum number of 512 KiB chunks.
	const target = 512 << 10
	if want := (len(input) + target - 1) / target; len(chunks) != want {
		t.Fatalf("got %d chunks, want %d", len(chunks), want)
	}
	for i, chunk := range chunks {
		if chunk.DecodedSize > target {
			t.Fatalf("fixture chunk %d exceeds the 512 KiB target: %d", i, chunk.DecodedSize)
		}
		if chunk.FrequencyProfile != chunks[0].FrequencyProfile {
			t.Fatalf("chunk %d changed frequency profile", i)
		}
	}
	for _, workers := range []int{2, 4, 8} {
		parallel, err := encodeWithWorkers(
			input, resources, FormatVersion, workers,
		)
		if err != nil || !bytes.Equal(parallel, serial) {
			t.Fatalf("%d encoding workers: bytes differ: %v", workers, err)
		}
		decoded, err := DecodeWithWorkers(parallel, workers)
		if err != nil || !bytes.Equal(decoded, input) {
			t.Fatalf("%d decoding workers: bytes differ: %v", workers, err)
		}
	}
	// This truncates a record after several full chunks have been submitted.
	truncated := input[:len(input)/2]
	for _, workers := range []int{1, 2, 8} {
		_, err := encodeWithWorkers(
			truncated, resources, FormatVersion, workers,
		)
		if err == nil {
			t.Fatalf("%d workers accepted a truncated MVD", workers)
		}
	}
}
