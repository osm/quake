package wad

import (
	"bytes"
	"errors"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/lump"
	"github.com/osm/quake/common/lump/typ"
)

var ErrIncorrectMagic = errors.New("incorrect magic")

const (
	headerSize   = 12
	entrySize    = 32
	filenameSize = 16
)

var magic = []byte("WAD2")

type Entry struct {
	Name        string
	Type        typ.Type
	Compression uint8
	Lump        lump.LumpData
}

type Wad struct {
	Entries []*Entry
}

func (w *Wad) Bytes() []byte {
	var offset uint32 = headerSize
	fbuf := buffer.New()

	for _, e := range w.Entries {
		lumpData := e.Lump.Bytes()
		fbuf.PutBytes(lumpData)
		offset += uint32(len(lumpData))
	}

	buf := buffer.New()
	buf.PutBytes(magic)
	buf.PutUint32(uint32(len(w.Entries)))
	buf.PutUint32(offset)

	buf.PutBytes(fbuf.Bytes())

	var currentOffset uint32 = headerSize
	for _, e := range w.Entries {
		lumpData := e.Lump.Bytes()
		size := uint32(len(lumpData))

		buf.PutUint32(currentOffset)
		buf.PutUint32(size)
		buf.PutUint32(size)
		buf.PutUint8(uint8(e.Type))
		buf.PutUint8(e.Compression)
		buf.PutUint16(0) // Padding

		var name [filenameSize]byte
		copy(name[:], e.Name)
		buf.PutBytes(name[:])

		currentOffset += size
	}

	return buf.Bytes()
}

func Parse(data []byte) (*Wad, error) {
	var wad Wad
	buf := buffer.New(buffer.WithData(data))

	m, err := buf.GetBytes(4)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(m, magic) {
		return nil, ErrIncorrectMagic
	}

	count, err := buf.GetUint32()
	if err != nil {
		return nil, err
	}

	dirOffset, err := buf.GetUint32()
	if err != nil {
		return nil, err
	}

	if err := buf.Seek(int(dirOffset)); err != nil {
		return nil, err
	}

	entries := make([]*Entry, count)
	for i := 0; i < int(count); i++ {
		offset, err := buf.GetUint32()
		if err != nil {
			return nil, err
		}

		size, err := buf.GetUint32()
		if err != nil {
			return nil, err
		}

		// Size in memory (same as size unless compression has been used)
		_, err = buf.GetUint32()
		if err != nil {
			return nil, err
		}

		lt, err := buf.GetUint8()
		if err != nil {
			return nil, err
		}
		ltype := typ.Type(lt)

		comp, err := buf.GetUint8()
		if err != nil {
			return nil, err
		}

		// Padding bytes
		if _, err := buf.GetUint16(); err != nil {
			return nil, err
		}

		name, err := buf.GetBytes(filenameSize)
		if err != nil {
			return nil, err
		}

		lumpData, err := buf.PeekBytesAt(int(offset), int(size))
		if err != nil {
			return nil, err
		}

		lump, err := lump.Parse(ltype, lumpData)
		if err != nil {
			return nil, err
		}

		entries[i] = &Entry{
			Name:        trim(name),
			Type:        ltype,
			Compression: comp,
			Lump:        lump,
		}
	}

	wad.Entries = entries
	return &wad, nil
}

func (w *Wad) GetEntry(name string) *Entry {
	for _, e := range w.Entries {
		if e.Name == name {
			return e
		}
	}

	return nil
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
