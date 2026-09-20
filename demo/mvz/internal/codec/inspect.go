package codec

import (
	"fmt"

	"github.com/osm/quake/demo/mvd"
	"github.com/osm/quake/packet/command"
)

// IsStructuredRecord reports whether the encoder recognizes a parsed record
// as structured data. It shares the encoder's recognition path.
func IsStructuredRecord(data *mvd.Data) (bool, error) {
	if data == nil {
		return false, fmt.Errorf("nil MVD record")
	}
	record, err := newRecord(data)
	return record.structured, err
}

// IsStructuredService reports whether the encoder recognizes a parsed service
// and its original bytes as a structured operation.
func IsStructuredService(parsed command.Command, raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	return newOperation(parsed, raw).structured()
}

// ChunkInfo describes the sizes and profile stored in one chunk's descriptors.
// DeflateEncodedSize includes the DEFLATE descriptor, as well as its streams.
type ChunkInfo struct {
	DecodedSize        int
	EncodedSize        int
	FrequencyProfile   byte
	RangeSize          int
	DeflateDecodedSize int
	DeflateEncodedSize int
}

// InspectChunks reads container and stream descriptors without decompressing
// data or verifying checksums. Use Decode to validate chunk data.
func InspectChunks(encoded []byte) ([]ChunkInfo, error) {
	if _, err := parseFileHeader(encoded); err != nil {
		return nil, err
	}
	var chunks []ChunkInfo
	remaining := encoded[headerSize:]
	for {
		header, end, err := parseChunkHeader(remaining)
		if err != nil {
			return nil, err
		}
		if end {
			if trailing := len(remaining) - endEntrySize; trailing != 0 {
				return nil, fmt.Errorf("MVZ has %d trailing bytes", trailing)
			}
			return chunks, nil
		}
		remaining = remaining[chunkHeaderSize:]
		payload, err := readChunkPayload(remaining, header)
		if err != nil {
			return nil, err
		}
		remaining = remaining[len(payload):]
		info, err := inspectChunk(header, payload)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, info)
	}
}

func inspectChunk(header containerChunkHeader, payload []byte) (ChunkInfo, error) {
	parts, err := splitChunkPayload(payload)
	if err != nil {
		return ChunkInfo{}, err
	}
	decodedSizes, err := readDecodedSizes(
		parts.deflateData, int(header.decodedSize),
	)
	if err != nil {
		return ChunkInfo{}, err
	}
	if _, err := readEncodedSizes(parts.deflateData); err != nil {
		return ChunkInfo{}, err
	}
	info := ChunkInfo{
		DecodedSize:        int(header.decodedSize),
		EncodedSize:        int(header.encodedSize),
		FrequencyProfile:   parts.profile,
		RangeSize:          len(parts.rangeData),
		DeflateEncodedSize: len(parts.deflateData),
	}
	for _, size := range decodedSizes {
		info.DeflateDecodedSize += size
	}
	return info, nil
}
