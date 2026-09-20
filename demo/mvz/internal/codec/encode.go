package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

func encodeWithWorkers(
	mvdData []byte,
	resources *RangeResources,
	version uint16,
	workers int,
) ([]byte, error) {
	encoder := rangeFileEncoder{
		resources: resources,
		workers:   workers,
	}
	if err := writeFileHeader(&encoder.out, version); err != nil {
		return nil, err
	}
	parseErr := parseIntoChunks(mvdData, encoder.writeChunk)
	// Pending chunks precede a parser failure in wire order. Finish them even
	// on failure, both to preserve error order and to join every worker.
	if encoder.chunks != nil {
		if err := encoder.chunks.finish(); err != nil {
			return nil, err
		}
	}
	if parseErr != nil {
		return nil, parseErr
	}
	if err := writeEndEntry(&encoder.out); err != nil {
		return nil, err
	}
	return encoder.out.Bytes(), nil
}

// Choose a profile on the first chunk so later chunks avoid repeating the trials.
type rangeFileEncoder struct {
	out               bytes.Buffer
	resources         *RangeResources
	selectedProfileID byte
	workers           int
	chunkIndex        int
	chunks            *chunkPipeline
}

func (encoder *rangeFileEncoder) writeChunk(chunk parsedRangeChunk) error {
	index := encoder.chunkIndex
	var err error
	if encoder.selectedProfileID == 0 || encoder.workers <= 1 {
		err = encoder.writeChunkNow(chunk, index)
	} else {
		err = encoder.submitChunk(chunk, index)
	}
	if err != nil {
		return err
	}
	encoder.chunkIndex++
	return nil
}

func (encoder *rangeFileEncoder) writeChunkNow(
	chunk parsedRangeChunk,
	index int,
) error {
	prepared, err := chunk.prepare()
	chunk.records = nil
	if err != nil {
		return err
	}
	if err := encodeChunk(
		&encoder.out,
		encoder.resources,
		&encoder.selectedProfileID,
		prepared,
		encoder.workers,
	); err != nil {
		return fmt.Errorf("encode MVZ chunk %d: %w", index, err)
	}
	return nil
}

func (encoder *rangeFileEncoder) submitChunk(
	chunk parsedRangeChunk,
	index int,
) error {
	if encoder.chunks == nil {
		encoder.chunks = newChunkPipeline(&encoder.out, encoder.workers)
	}
	// Keep the selector local: concurrent jobs must not share its writable pointer.
	profileID := encoder.selectedProfileID
	return encoder.chunks.submit(len(chunk.original), func() ([]byte, error) {
		prepared, err := chunk.prepare()
		// Release the parsed command tree before allocating compression buffers.
		chunk.records = nil
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if err := encodeChunk(
			&out, encoder.resources, &profileID, prepared, 1,
		); err != nil {
			return nil, fmt.Errorf("encode MVZ chunk %d: %w", index, err)
		}
		return out.Bytes(), nil
	})
}

func encodeChunk(
	dst io.Writer,
	resources *RangeResources,
	selectedProfileID *byte,
	chunk rangeChunkInput,
	workers int,
) error {
	input, err := compressLiterals(chunk)
	if err != nil {
		return err
	}
	if *selectedProfileID != 0 {
		profile, ok := findProfile(resources.profiles, *selectedProfileID)
		if !ok {
			return fmt.Errorf(
				"selected MVZ frequency profile %d is unavailable",
				*selectedProfileID,
			)
		}
		rangeData, err := encodeRangeStream(input, profile.models)
		if err != nil {
			return err
		}
		return writeChunk(dst, input, rangeData, profile.id)
	}

	rangeStreams, err := tryProfiles(
		input,
		resources,
		workers,
	)
	if err != nil {
		return err
	}
	selectedIndex := 0
	for i := 1; i < len(rangeStreams); i++ {
		if len(rangeStreams[i]) < len(rangeStreams[selectedIndex]) {
			selectedIndex = i
		}
	}
	profileID := resources.profiles[selectedIndex].id
	*selectedProfileID = profileID
	return writeChunk(
		dst, input, rangeStreams[selectedIndex], profileID,
	)
}

// Original bytes are still needed for the decoded size and checksum.
type rangeProfileInput struct {
	records        []rangeRecord
	trailing       []byte
	original       []byte
	encodedDeflate []byte
}

// The literal streams do not depend on the selected range profile. Compress
// them once and share the result across the first chunk's profile trials.
func compressLiterals(chunk rangeChunkInput) (*rangeProfileInput, error) {
	data := collectLiterals(chunk.records, chunk.trailing)
	input := &rangeProfileInput{
		records:  chunk.records,
		trailing: chunk.trailing,
		original: chunk.original,
	}
	var err error
	input.encodedDeflate, err = data.encode()
	if err != nil {
		return nil, err
	}
	return input, nil
}

func writeChunk(
	dst io.Writer,
	input *rangeProfileInput,
	rangeData []byte,
	profile byte,
) error {
	payloadSize := rangeDescriptorSize + len(rangeData) + len(input.encodedDeflate)
	if err := writeChunkHeader(
		dst, payloadSize, input.original,
	); err != nil {
		return err
	}
	var descriptor [rangeDescriptorSize]byte
	binary.LittleEndian.PutUint32(descriptor[0:4], uint32(len(rangeData)))
	descriptor[4] = profile
	parts := [...][]byte{descriptor[:], rangeData, input.encodedDeflate}
	for _, part := range parts {
		if _, err := dst.Write(part); err != nil {
			return fmt.Errorf("write MVZ chunk payload: %w", err)
		}
	}
	return nil
}
