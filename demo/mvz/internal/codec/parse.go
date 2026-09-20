package codec

import (
	"bytes"
	"fmt"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/demo/mvd"
)

// The target is encoder policy, not the decoder's absolute limit on chunk size.
const rangeChunkTargetSize = 512 << 10

// Parsing must be sequential because records can change protocol state.
// Each chunk owns copies of the parsed records, so workers can prepare its
// operations after the parser has advanced.
type parsedRangeChunk struct {
	records     []mvd.Data
	trailing    []byte
	original    []byte
	firstRecord int
}

func parseIntoChunks(
	input []byte,
	visit func(parsedRangeChunk) error,
) error {
	if visit == nil {
		return fmt.Errorf("nil MVZ chunk visitor")
	}
	builder := &rangeChunkBuilder{input: input, visit: visit}
	trailing, err := mvd.ParseRecords(context.New(), input, builder.addRecord)
	if err != nil {
		if builder.visitErr != nil {
			return builder.visitErr
		}
		// Complete records in the unfinished chunk precede the parse failure.
		// Preserve their preparation errors without submitting partial output.
		if _, prepareErr := builder.chunk(nil).prepare(); prepareErr != nil {
			return prepareErr
		}
		return fmt.Errorf("parse MVD input: %w", err)
	}
	if !bytes.Equal(trailing, input[builder.offset:]) {
		return fmt.Errorf("MVD parser lost trailing bytes")
	}
	return builder.flush(trailing)
}

type rangeChunkBuilder struct {
	input       []byte
	visit       func(parsedRangeChunk) error
	visitErr    error
	records     []mvd.Data
	chunkSize   int
	chunkStart  int
	recordIndex int
	offset      int
}

func (builder *rangeChunkBuilder) addRecord(data *mvd.Data) error {
	if err := builder.validateOriginalBytes(data); err != nil {
		return err
	}
	recordSize := len(data.OriginalBytes())
	if recordSize < 2 {
		return fmt.Errorf(
			"MVD record %d is shorter than its header",
			builder.recordIndex,
		)
	}
	if recordSize > maxChunkSize {
		return fmt.Errorf(
			"MVD record %d size %d exceeds MVZ chunk limit %d",
			builder.recordIndex,
			recordSize,
			maxChunkSize,
		)
	}
	if builder.chunkSize != 0 &&
		builder.chunkSize+recordSize > rangeChunkTargetSize {
		if err := builder.flush(nil); err != nil {
			return err
		}
	}
	builder.offset += recordSize
	builder.records = append(builder.records, *data)
	builder.chunkSize += recordSize
	builder.recordIndex++
	return nil
}

func (builder *rangeChunkBuilder) validateOriginalBytes(data *mvd.Data) error {
	raw := data.OriginalBytes()
	if len(raw) == 0 || len(raw) > len(builder.input)-builder.offset ||
		!bytes.Equal(raw, builder.input[builder.offset:builder.offset+len(raw)]) {
		return fmt.Errorf("MVD parser lost record %d bytes", builder.recordIndex)
	}
	return nil
}

func (builder *rangeChunkBuilder) flush(trailing []byte) error {
	if len(builder.records) == 0 && len(trailing) == 0 {
		return nil
	}
	if err := validateChunkSize(
		"decoded",
		uint64(builder.chunkSize)+uint64(len(trailing)),
	); err != nil {
		return err
	}
	chunk := builder.chunk(trailing)
	// Let the visitor release parsed records before compression. The builder
	// must not keep the command tree alive during profile trials on the first chunk.
	builder.records = nil
	builder.chunkSize = 0
	builder.chunkStart = builder.offset + len(trailing)
	if err := builder.visit(chunk); err != nil {
		builder.visitErr = err
		return err
	}
	return nil
}

func (builder *rangeChunkBuilder) chunk(trailing []byte) parsedRangeChunk {
	return parsedRangeChunk{
		records:     builder.records,
		trailing:    trailing,
		original:    builder.input[builder.chunkStart : builder.offset+len(trailing)],
		firstRecord: builder.recordIndex - len(builder.records),
	}
}
