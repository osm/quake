package buffer

import (
	"encoding/binary"
	"math"
	"strings"
)

func (b *Buffer) ReadByte() (byte, error) {
	if b.off+1 > len(b.buf) {
		return 0, ErrBadRead
	}

	r := b.buf[b.off]
	b.off += 1
	return r, nil
}

func (b *Buffer) GetBytes(n int) ([]byte, error) {
	if b.off+n > len(b.buf) {
		return nil, ErrBadRead
	}

	r := b.buf[b.off : b.off+n]
	b.off += n
	return r, nil
}

func (b *Buffer) GetUint8() (uint8, error) {
	if b.off+1 > len(b.buf) {
		return 0, ErrBadRead
	}

	r := b.buf[b.off]
	b.off++
	return r, nil
}

func (b *Buffer) GetInt8() (int8, error) {
	r, err := b.GetUint8()
	if err != nil {
		return 0, err
	}

	return int8(r), nil
}

func (b *Buffer) GetInt16() (int16, error) {
	r, err := b.GetUint16()
	if err != nil {
		return 0, err
	}

	return int16(r), nil
}

func (b *Buffer) GetUint16() (uint16, error) {
	if b.off+2 > len(b.buf) {
		return 0, ErrBadRead
	}

	r := binary.LittleEndian.Uint16(b.buf[b.off : b.off+2])
	b.off += 2
	return r, nil
}

func (b *Buffer) GetInt32() (int32, error) {
	r, err := b.GetUint32()
	if err != nil {
		return 0, err
	}

	return int32(r), nil
}

func (b *Buffer) GetUint32() (uint32, error) {
	if b.off+4 > len(b.buf) {
		return 0, ErrBadRead
	}

	r := binary.LittleEndian.Uint32(b.buf[b.off : b.off+4])
	b.off += 4
	return r, nil
}

func (b *Buffer) GetFloat32() (float32, error) {
	r, err := b.GetUint32()
	if err != nil {
		return 0, err
	}

	return math.Float32frombits(r), nil
}

func (b *Buffer) GetAngle8() (float32, error) {
	r, err := b.ReadByte()
	if err != nil {
		return 0, err
	}

	return float32(int8(r)) * (360.0 / 256), nil
}

func (b *Buffer) GetAngle16() (float32, error) {
	r, err := b.GetUint16()
	if err != nil {
		return 0, err
	}

	return float32(r) * (360.0 / 65536), nil
}

func (b *Buffer) GetCoord16() (float32, error) {
	r, err := b.GetInt16()
	if err != nil {
		return 0, err
	}

	return float32(r) / 8.0, nil
}

func (b *Buffer) GetCoord32() (float32, error) {
	return b.GetFloat32()
}

func (b *Buffer) GetString() (string, error) {
	var str strings.Builder

	for b.off < b.Len() {
		r, err := b.ReadByte()
		if err != nil {
			return "", err
		}

		if r == 0xff {
			continue
		}

		if r == 0x00 {
			break
		}

		str.WriteByte(r)
	}

	return str.String(), nil
}

func (b *Buffer) PeekInt32() (int32, error) {
	if b.off+4 > len(b.buf) {
		return 0, ErrBadRead
	}

	return int32(binary.LittleEndian.Uint32(b.buf[b.off : b.off+4])), nil
}

func (b *Buffer) PeekBytes(n int) ([]byte, error) {
	if b.off+n > len(b.buf) {
		return nil, ErrBadRead
	}

	return b.buf[b.off : b.off+n], nil
}

func (b *Buffer) PeekBytesAt(off, size int) ([]byte, error) {
	if off+size > len(b.buf) {
		return nil, ErrBadRead
	}

	return b.buf[off : off+size], nil
}

func (b *Buffer) Skip(n int) error {
	if b.off+n > len(b.buf) {
		return ErrBadRead
	}

	b.off += n
	return nil
}

func (b *Buffer) Seek(n int) error {
	if n > len(b.buf) {
		return ErrBadRead
	}

	b.off = n
	return nil
}
