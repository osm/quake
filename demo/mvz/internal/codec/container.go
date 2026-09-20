package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

const rangeDescriptorSize = 5

type containerChunkHeader struct {
	decodedSize uint32
	checksum    uint32
	encodedSize uint32
}

func (header containerChunkHeader) validate() error {
	if err := validateChunkSize("decoded", uint64(header.decodedSize)); err != nil {
		return err
	}
	return validateChunkSize("encoded", uint64(header.encodedSize))
}

func validateChunkSize(kind string, size uint64) error {
	if size == 0 || size > maxChunkSize {
		return fmt.Errorf("invalid MVZ %s chunk size %d", kind, size)
	}
	return nil
}

func parseFileHeader(data []byte) (uint16, error) {
	if len(data) < headerSize {
		return 0, fmt.Errorf("read MVZ header: %w", io.ErrUnexpectedEOF)
	}
	if !bytes.Equal(data[:4], magic[:]) {
		return 0, fmt.Errorf("invalid MVZ magic")
	}
	return binary.LittleEndian.Uint16(data[4:6]), nil
}

func writeFileHeader(dst io.Writer, version uint16) error {
	var header [headerSize]byte
	copy(header[:4], magic[:])
	binary.LittleEndian.PutUint16(header[4:6], version)
	if _, err := dst.Write(header[:]); err != nil {
		return fmt.Errorf("write MVZ header: %w", err)
	}
	return nil
}

func writeChunkHeader(
	dst io.Writer,
	encodedSize int,
	decoded []byte,
) error {
	if err := validateChunkSize("decoded", uint64(len(decoded))); err != nil {
		return err
	}
	if err := validateChunkSize("encoded", uint64(encodedSize)); err != nil {
		return err
	}
	var header [chunkHeaderSize]byte
	header[0] = entryTypeData
	binary.LittleEndian.PutUint32(header[1:5], uint32(len(decoded)))
	binary.LittleEndian.PutUint32(header[5:9], crc32.ChecksumIEEE(decoded))
	binary.LittleEndian.PutUint32(header[9:13], uint32(encodedSize))
	if _, err := dst.Write(header[:]); err != nil {
		return fmt.Errorf("write MVZ chunk header: %w", err)
	}
	return nil
}

func parseChunkHeader(
	data []byte,
) (containerChunkHeader, bool, error) {
	if len(data) == 0 {
		return containerChunkHeader{}, false, fmt.Errorf(
			"read MVZ entry type: %w", io.ErrUnexpectedEOF,
		)
	}
	switch data[0] {
	case entryTypeEnd:
		return containerChunkHeader{}, true, nil
	case entryTypeData:
	default:
		return containerChunkHeader{}, false, fmt.Errorf(
			"unknown MVZ entry type %d", data[0],
		)
	}
	if len(data) < chunkHeaderSize {
		return containerChunkHeader{}, false, fmt.Errorf(
			"read MVZ chunk header: %w", io.ErrUnexpectedEOF,
		)
	}
	return containerChunkHeader{
		decodedSize: binary.LittleEndian.Uint32(data[1:5]),
		checksum:    binary.LittleEndian.Uint32(data[5:9]),
		encodedSize: binary.LittleEndian.Uint32(data[9:13]),
	}, false, nil
}

// The payload shares the input's storage and must not be modified.
func readChunkPayload(
	data []byte,
	header containerChunkHeader,
) ([]byte, error) {
	if err := header.validate(); err != nil {
		return nil, err
	}
	size := int(header.encodedSize)
	if len(data) < size {
		return nil, fmt.Errorf("read MVZ chunk payload: %w", io.ErrUnexpectedEOF)
	}
	return data[:size], nil
}

// Check the bound before converting to int to avoid overflow when int has 32 bits.
func readBoundedSize(data []byte, limit int) (int, bool) {
	value := binary.LittleEndian.Uint32(data)
	if limit < 0 || uint64(value) > uint64(limit) {
		return 0, false
	}
	return int(value), true
}

type rangeChunkPayload struct {
	profile     byte
	rangeData   []byte
	deflateData []byte
}

func splitChunkPayload(payload []byte) (rangeChunkPayload, error) {
	if len(payload) < rangeDescriptorSize {
		return rangeChunkPayload{}, fmt.Errorf(
			"MVZ range stream is shorter than its header",
		)
	}
	rangeSize, ok := readBoundedSize(
		payload[0:4], len(payload)-rangeDescriptorSize,
	)
	if !ok || rangeSize == 0 {
		return rangeChunkPayload{}, fmt.Errorf("invalid MVZ range stream size")
	}
	rangeStart := rangeDescriptorSize
	rangeEnd := rangeStart + rangeSize
	return rangeChunkPayload{
		profile:     payload[4],
		rangeData:   payload[rangeStart:rangeEnd],
		deflateData: payload[rangeEnd:],
	}, nil
}

func writeEndEntry(dst io.Writer) error {
	if _, err := dst.Write([]byte{entryTypeEnd}); err != nil {
		return fmt.Errorf("write MVZ end entry: %w", err)
	}
	return nil
}
