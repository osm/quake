package pak

import (
	"bytes"
	"errors"

	"github.com/osm/quake/common/buffer"
)

var ErrIncorrectMagic = errors.New("incorrect magic")

const (
	headerOffset   = 12
	fileHeaderSize = 64
	filenameSize   = 56
)

var magic = []byte("PACK")

type Pak struct {
	Files []*File
}

type File struct {
	offset uint32

	Path string
	Data []byte
}

func (p *Pak) Bytes() []byte {
	var offset uint32 = headerOffset

	fbuf := buffer.New()
	for _, f := range p.Files {
		f.offset = offset
		fbuf.PutBytes(f.Data)
		offset += uint32(len(f.Data))
	}

	buf := buffer.New()
	buf.PutBytes(magic)
	buf.PutUint32(offset)
	buf.PutUint32(uint32(fileHeaderSize * len(p.Files)))
	buf.PutBytes(fbuf.Bytes())

	for _, f := range p.Files {
		var path [filenameSize]byte
		copy(path[:], f.Path)
		buf.PutBytes(path[:])
		buf.PutUint32(f.offset)
		buf.PutUint32(uint32(len(f.Data)))
	}

	return buf.Bytes()
}

func Parse(data []byte) (*Pak, error) {
	var pak Pak
	buf := buffer.New(buffer.WithData(data))

	m, err := buf.GetBytes(4)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(m, magic) {
		return nil, ErrIncorrectMagic
	}

	offset, err := buf.GetUint32()
	if err != nil {
		return nil, err
	}

	size, err := buf.GetUint32()
	if err != nil {
		return nil, err
	}

	if err := buf.Seek(int(offset)); err != nil {
		return nil, err
	}

	count := int(size / fileHeaderSize)
	files := make([]*File, count)

	for i := 0; i < count; i++ {
		name, err := buf.GetBytes(filenameSize)
		if err != nil {
			return nil, err
		}

		offset, err := buf.GetUint32()
		if err != nil {
			return nil, err
		}

		size, err := buf.GetUint32()
		if err != nil {
			return nil, err
		}

		data, err := buf.PeekBytesAt(int(offset), int(size))
		if err != nil {
			return nil, err
		}

		files[i] = &File{
			Path: trim(name),
			Data: data,
		}
	}
	pak.Files = files

	return &pak, nil
}

func trim(b []byte) string {
	var i int

	for i = 0; i < len(b); i++ {
		if b[i] == 0x00 {
			break
		}
	}

	return string(b[:i])
}
