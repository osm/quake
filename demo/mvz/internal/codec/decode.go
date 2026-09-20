package codec

import (
	"bytes"
	"fmt"
	"hash/crc32"

	"github.com/osm/quake/demo/mvz/internal/rangecoder"
)

func decodeChunks(
	remaining []byte,
	profiles []rangeProfile,
	workers int,
) ([]byte, error) {
	var out bytes.Buffer
	var chunks *chunkPipeline
	var readErr error
	for {
		header, end, err := parseChunkHeader(remaining)
		if err != nil {
			readErr = err
			break
		}
		if end {
			if trailing := len(remaining) - endEntrySize; trailing != 0 {
				readErr = fmt.Errorf("%d trailing bytes after MVZ end entry", trailing)
			}
			break
		}
		remaining = remaining[chunkHeaderSize:]
		payload, err := readChunkPayload(remaining, header)
		if err != nil {
			readErr = err
			break
		}
		remaining = remaining[len(payload):]
		// Avoid worker setup for files with one chunk. The remaining entry is
		// still read and validated normally on the next iteration.
		if workers <= 1 || (chunks == nil && len(remaining) == endEntrySize) {
			decoded, err := decodeChunk(header, payload, profiles)
			if err != nil {
				readErr = err
				break
			}
			out.Write(decoded)
			continue
		}
		if chunks == nil {
			chunks = newChunkPipeline(&out, workers)
		}
		size := len(payload) + int(header.decodedSize)
		if err := chunks.submit(size, func() ([]byte, error) {
			return decodeChunk(header, payload, profiles)
		}); err != nil {
			readErr = err
			break
		}
	}
	// Earlier chunk failures take precedence over a later framing error.
	if chunks != nil {
		if err := chunks.finish(); err != nil {
			return nil, err
		}
	}
	if readErr != nil {
		return nil, readErr
	}
	if out.Len() == 0 {
		return []byte{}, nil
	}
	return out.Bytes(), nil
}

func decodeChunk(
	header containerChunkHeader,
	payload []byte,
	profiles []rangeProfile,
) ([]byte, error) {
	decoded, err := decodeChunkPayload(payload, int(header.decodedSize), profiles)
	if err != nil {
		return nil, err
	}
	if got := crc32.ChecksumIEEE(decoded); got != header.checksum {
		return nil, fmt.Errorf(
			"MVZ chunk checksum %#08x, want %#08x",
			got,
			header.checksum,
		)
	}
	return decoded, nil
}

// Successful range decoding does not verify integrity. The caller checks the CRC.
func decodeChunkPayload(
	payload []byte,
	decodedSize int,
	profiles []rangeProfile,
) ([]byte, error) {
	if err := validateChunkSize("decoded", uint64(decodedSize)); err != nil {
		return nil, err
	}
	parts, err := splitChunkPayload(payload)
	if err != nil {
		return nil, err
	}
	selected, ok := findProfile(profiles, parts.profile)
	if !ok {
		return nil, fmt.Errorf("invalid MVZ frequency profile %d", parts.profile)
	}
	decodedDeflate, err := decodeLiterals(parts.deflateData, decodedSize)
	if err != nil {
		return nil, err
	}

	// Every chunk starts with zero prediction history and fresh literal cursors.
	// Records consume the literals as directed by range symbols; finish checks
	// exact literal consumption and reconstructed size.
	decoder := rangecoder.NewDecoder(parts.rangeData)
	chunk := rangeChunkDecoder{
		decoder:     decoder,
		models:      selected.models,
		literals:    decodedDeflate.general,
		textStreams: decodedDeflate.text,
		decodedSize: decodedSize,
		out:         make([]byte, 0, decodedSize),
	}
	if err := chunk.decodeRecords(); err != nil {
		return nil, err
	}
	if err := chunk.finish(); err != nil {
		return nil, err
	}
	return chunk.out, nil
}
