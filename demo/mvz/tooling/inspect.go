// Package tooling provides MVZ inspection, profiling, and training helpers.
// Use package mvz for ordinary encoding and decoding.
package tooling

import (
	"github.com/osm/quake/demo/mvd"
	"github.com/osm/quake/demo/mvz/internal/codec"
	"github.com/osm/quake/packet/command"
)

// ChunkInfo describes the sizes and profile stored in one chunk's descriptors.
// DeflateEncodedSize includes the DEFLATE descriptor, as well as its streams.
type ChunkInfo = codec.ChunkInfo

// InspectChunks reads container and stream descriptors without decompressing
// data or verifying checksums. Use mvz.Decode to validate chunk data.
func InspectChunks(encoded []byte) ([]ChunkInfo, error) {
	return codec.InspectChunks(encoded)
}

// FrequencyProfileIDs returns the stored frequency profile ID for every chunk.
func FrequencyProfileIDs(encoded []byte) ([]byte, error) {
	chunks, err := InspectChunks(encoded)
	if err != nil {
		return nil, err
	}
	var profiles []byte
	for _, chunk := range chunks {
		profiles = append(profiles, chunk.FrequencyProfile)
	}
	return profiles, nil
}

// IsStructuredRecord reports whether the encoder recognizes a parsed record
// as structured data. It shares the encoder's recognition path.
func IsStructuredRecord(data *mvd.Data) (bool, error) {
	return codec.IsStructuredRecord(data)
}

// IsStructuredService reports whether the encoder recognizes a parsed service
// and its original bytes as a structured operation.
func IsStructuredService(parsed command.Command, raw []byte) bool {
	return codec.IsStructuredService(parsed, raw)
}
