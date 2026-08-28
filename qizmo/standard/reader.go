package standard

import (
	"bytes"
	"encoding/binary"
	"io"
)

type reader struct {
	*bytes.Reader
}

func newReader(b []byte) *reader {
	return &reader{bytes.NewReader(b)}
}

func (r *reader) readN(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(r, b)
	return b, err
}

func (r *reader) readUint16() (uint16, error) {
	var v uint16
	err := binary.Read(r, binary.LittleEndian, &v)
	return v, err
}
