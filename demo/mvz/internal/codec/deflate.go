package codec

import (
	"bytes"
	"compress/flate"
	"fmt"
	"io"
)

func encodeDeflate(data []byte) ([]byte, error) {
	var compressed bytes.Buffer
	zw, err := flate.NewWriter(&compressed, flate.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("create MVZ DEFLATE compressor: %w", err)
	}
	if _, err := zw.Write(data); err != nil {
		return nil, fmt.Errorf("compress MVZ DEFLATE stream: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finish MVZ DEFLATE stream: %w", err)
	}
	return compressed.Bytes(), nil
}

func decodeDeflate(payload []byte, decodedSize int) ([]byte, error) {
	src := bytes.NewReader(payload)
	zr := flate.NewReader(src)
	decoded := make([]byte, decodedSize)
	_, err := io.ReadFull(zr, decoded)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	// Read once beyond the advertised output to require the DEFLATE end marker
	// and reject extra decoded bytes. The compressed boundary is checked below.
	var extra [1]byte
	extraBytes, extraErr := io.ReadFull(zr, extra[:])
	closeErr := zr.Close()
	if err != nil {
		return nil, fmt.Errorf("decompress MVZ chunk: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close MVZ decompressor: %w", closeErr)
	}
	if extraBytes != 0 {
		return nil, fmt.Errorf(
			"MVZ decoded chunk exceeds advertised size %d",
			decodedSize,
		)
	}
	if extraErr != io.EOF {
		return nil, fmt.Errorf("decompress MVZ chunk: %w", extraErr)
	}
	if src.Len() != 0 {
		return nil, fmt.Errorf(
			"MVZ DEFLATE stream has %d trailing compressed bytes",
			src.Len(),
		)
	}
	return decoded, nil
}
