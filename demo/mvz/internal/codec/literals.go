package codec

import (
	"encoding/binary"
	"fmt"
)

// Every chunk has four raw DEFLATE streams in this wire order: print,
// stufftext, centerprint, general literals. The descriptor stores their encoded and
// decoded sizes. All four streams must end at exactly those boundaries.
const (
	deflateStreamCount    = rangeTextStreamCount + 1
	deflateGeneralStream  = deflateStreamCount - 1
	deflateSizeTableSize  = deflateStreamCount * 4
	deflateDescriptorSize = deflateSizeTableSize * 2
)

type deflateData struct {
	general []byte
	text    [rangeTextStreamCount][]byte
}

func (data *deflateData) appendRecordLiterals(record rangeRecord) {
	if !record.structured {
		data.general = append(data.general, record.payload...)
		return
	}
	data.general = append(data.general, record.prefix...)
	for _, operation := range record.operations {
		switch {
		case operation.text != nil:
			stream := textStreamIndex(operation.text.opcode)
			data.text[stream] = append(data.text[stream], operation.text.payload...)
		case !operation.structured():
			data.general = append(data.general, operation.raw...)
		}
	}
}

func (data *deflateData) encode() ([]byte, error) {
	streams := [deflateStreamCount][]byte{
		data.text[rangeTextStreamPrint],
		data.text[rangeTextStreamStuffText],
		data.text[rangeTextStreamCenterPrint],
		data.general,
	}
	var compressed [deflateStreamCount][]byte
	total := deflateDescriptorSize
	for i, stream := range streams {
		var err error
		compressed[i], err = encodeDeflate(stream)
		if err != nil {
			return nil, err
		}
		total += len(compressed[i])
	}
	out := make([]byte, deflateDescriptorSize, total)
	for i := range compressed {
		binary.LittleEndian.PutUint32(out[i*4:i*4+4], uint32(len(compressed[i])))
	}
	for i, stream := range streams {
		offset := deflateSizeTableSize + i*4
		binary.LittleEndian.PutUint32(out[offset:offset+4], uint32(len(stream)))
	}
	for _, stream := range compressed {
		out = append(out, stream...)
	}
	return out, nil
}

func collectLiterals(records []rangeRecord, trailing []byte) deflateData {
	var data deflateData
	for _, record := range records {
		data.appendRecordLiterals(record)
	}
	data.general = append(data.general, trailing...)
	return data
}

func decodeLiterals(
	payload []byte,
	decodedLimit int,
) (deflateData, error) {
	encodedSizes, err := readEncodedSizes(payload)
	if err != nil {
		return deflateData{}, err
	}
	decodedSizes, err := readDecodedSizes(payload, decodedLimit)
	if err != nil {
		return deflateData{}, err
	}
	return inflateStreams(payload, encodedSizes, decodedSizes)
}

func readDecodedSizes(
	payload []byte,
	decodedLimit int,
) ([deflateStreamCount]int, error) {
	var sizes [deflateStreamCount]int
	if len(payload) < deflateDescriptorSize {
		return sizes, fmt.Errorf(
			"MVZ DEFLATE data is shorter than its descriptor",
		)
	}
	total := 0
	for stream := range sizes {
		offset := deflateSizeTableSize + stream*4
		size, ok := readBoundedSize(
			payload[offset:offset+4],
			decodedLimit-total,
		)
		if !ok {
			return sizes, fmt.Errorf("invalid MVZ DEFLATE sizes")
		}
		sizes[stream] = size
		total += size
	}
	return sizes, nil
}

func readEncodedSizes(payload []byte) ([deflateStreamCount]int, error) {
	var sizes [deflateStreamCount]int
	if len(payload) < deflateDescriptorSize {
		return sizes, fmt.Errorf(
			"MVZ DEFLATE data is shorter than its descriptor",
		)
	}
	total := deflateDescriptorSize
	for stream := range sizes {
		offset := stream * 4
		size, ok := readBoundedSize(payload[offset:offset+4], len(payload)-total)
		if !ok {
			return sizes, fmt.Errorf("invalid MVZ DEFLATE sizes")
		}
		sizes[stream] = size
		total += size
	}
	if total != len(payload) {
		return sizes, fmt.Errorf("invalid MVZ DEFLATE sizes")
	}
	return sizes, nil
}

func inflateStreams(
	payload []byte,
	encodedSizes [deflateStreamCount]int,
	decodedSizes [deflateStreamCount]int,
) (deflateData, error) {
	var literals deflateData
	offset := deflateDescriptorSize
	for stream, encodedSize := range encodedSizes {
		end := offset + encodedSize
		decoded, err := decodeDeflate(payload[offset:end], decodedSizes[stream])
		if err != nil {
			return deflateData{}, fmt.Errorf(
				"decode MVZ DEFLATE stream %d: %w",
				stream,
				err,
			)
		}
		if stream == deflateGeneralStream {
			literals.general = decoded
		} else {
			literals.text[stream] = decoded
		}
		offset = end
	}
	return literals, nil
}
