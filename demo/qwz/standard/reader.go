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

func (r *reader) ReadN(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(r, b)
	return b, err
}

func (r *reader) ReadU16() (uint16, error) {
	var v uint16
	err := binary.Read(r, binary.LittleEndian, &v)
	return v, err
}

func (r *reader) ReadString() ([]byte, error) {
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		buf = append(buf, b)
		if b == 0 {
			return buf, nil
		}
	}
}
