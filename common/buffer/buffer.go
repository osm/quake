package buffer

import (
	"errors"
)

var ErrBadRead = errors.New("bad read")

type Buffer struct {
	buf []byte
	off int
}

func New(opts ...Option) *Buffer {
	b := &Buffer{}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

func (b *Buffer) Len() int {
	return len(b.buf)
}

func (b *Buffer) Off() int {
	return b.off
}

func (b *Buffer) Bytes() []byte {
	return b.buf
}
