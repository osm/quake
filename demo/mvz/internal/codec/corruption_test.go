package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strconv"
	"testing"
)

func TestCompatibilityVectorRejectsTruncation(t *testing.T) {
	encoded := readCompatibilityVector(t, "version-1-profile-1")
	payloadStart, payloadEnd := compatibilityPayloadBounds(t, encoded)
	points := []int{
		0,
		1,
		headerSize - 1,
		headerSize,
		headerSize + 1,
		payloadStart - 1,
		payloadStart,
		payloadStart + 1,
		payloadStart + (payloadEnd-payloadStart)/2,
		payloadEnd - 1,
		payloadEnd,
	}
	for _, length := range points {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			data := encoded[:length]
			if _, err := Decode(data); !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("decode truncation at %d: got %v, want unexpected EOF", length, err)
			}
		})
	}
}

func TestCompatibilityVectorRejectsCorruption(t *testing.T) {
	encoded := readCompatibilityVector(t, "version-1-profile-1")
	payloadStart, payloadEnd := compatibilityPayloadBounds(t, encoded)
	rangeSize := int(binary.LittleEndian.Uint32(encoded[payloadStart:]))
	rangeStart := payloadStart + rangeDescriptorSize
	literalStart := rangeStart + rangeSize
	literalSize := binary.LittleEndian.Uint32(encoded[literalStart+deflateSizeTableSize:])

	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{name: "bad magic", mutate: setByte(0, 'X')},
		{name: "bad version", mutate: setByte(4, 0xff)},
		{name: "unknown entry type", mutate: setByte(headerSize, 0xff)},
		{name: "retired raw entry type", mutate: setByte(headerSize, 2)},
		{name: "retired fallback entry type", mutate: setByte(headerSize, 3)},
		{name: "zero decoded size", mutate: setUint32(headerSize+1, 0)},
		{name: "large decoded size", mutate: setUint32(headerSize+1, maxChunkSize+1)},
		{name: "checksum", mutate: flipByte(headerSize + 5)},
		{name: "zero encoded size", mutate: setUint32(headerSize+9, 0)},
		{name: "large encoded size", mutate: setUint32(headerSize+9, maxChunkSize+1)},
		{name: "zero range size", mutate: setUint32(payloadStart, 0)},
		{name: "large range size", mutate: setUint32(payloadStart, math.MaxUint32)},
		{name: "invalid profile", mutate: setByte(payloadStart+4, 0)},
		{name: "large deflate size", mutate: setUint32(literalStart, math.MaxUint32)},
		{name: "large literal size", mutate: setUint32(literalStart+16, math.MaxUint32)},
		{
			name:   "short deflate output",
			mutate: setUint32(literalStart+deflateSizeTableSize, literalSize+1),
		},
		{name: "range data", mutate: flipByte(rangeStart + rangeSize/2)},
		{name: "deflate data", mutate: flipByte(literalStart + (payloadEnd-literalStart)/2)},
		{name: "end entry", mutate: setByte(payloadEnd, 0xff)},
		{name: "trailing data", mutate: func(data []byte) []byte {
			return append(data, 1)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			corrupt := test.mutate(append([]byte(nil), encoded...))
			decoded, err := Decode(corrupt)
			if err == nil || errors.Is(err, io.EOF) {
				t.Fatalf("decode corruption: got %v, want an error", err)
			}
			if decoded != nil {
				t.Fatal("decoder returned output from a corrupt file")
			}
		})
	}
}

func TestDeflateBounds(t *testing.T) {
	decoded := []byte("literal data")
	encoded, err := encodeDeflate(decoded)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		data []byte
		size int
	}{
		{name: "short output", data: encoded, size: len(decoded) + 1},
		{name: "long output", data: encoded, size: len(decoded) - 1},
		{
			name: "trailing bytes",
			data: append(bytes.Clone(encoded), 0xff), size: len(decoded),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeDeflate(test.data, test.size); err == nil {
				t.Fatal("DEFLATE decoder accepted invalid stream bounds")
			}
		})
	}
}

func compatibilityPayloadBounds(t testing.TB, encoded []byte) (int, int) {
	t.Helper()
	if len(encoded) < headerSize+chunkHeaderSize {
		t.Fatalf("compatibility vector is too short: %d", len(encoded))
	}
	start := headerSize + chunkHeaderSize
	size := int(binary.LittleEndian.Uint32(encoded[headerSize+9:]))
	end := start + size
	if end+endEntrySize != len(encoded) {
		t.Fatalf("unexpected compatibility vector size %d", len(encoded))
	}
	return start, end
}

func setByte(offset int, value byte) func([]byte) []byte {
	return func(data []byte) []byte {
		data[offset] = value
		return data
	}
}

func flipByte(offset int) func([]byte) []byte {
	return func(data []byte) []byte {
		data[offset] ^= 0x40
		return data
	}
}

func setUint32(offset int, value uint32) func([]byte) []byte {
	return func(data []byte) []byte {
		binary.LittleEndian.PutUint32(data[offset:], value)
		return data
	}
}
