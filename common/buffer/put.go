package buffer

import (
	"encoding/binary"
	"math"
)

func (b *Buffer) PutByte(v byte) {
	b.off += 1
	b.buf = append(b.buf, v)
}

func (b *Buffer) PutBytes(v []byte) {
	b.off += len(v)
	b.buf = append(b.buf, v...)
}

func (b *Buffer) PutInt16(v int16) {
	b.PutUint16(uint16(v))
}

func (b *Buffer) PutUint16(v uint16) {
	b.off += 2

	tmp := make([]byte, 2)
	binary.LittleEndian.PutUint16(tmp, v)
	b.buf = append(b.buf, tmp...)
}

func (b *Buffer) PutInt32(v int32) {
	b.PutUint32(uint32(v))
}

func (b *Buffer) PutUint32(v uint32) {
	b.off += 4

	tmp := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmp, v)
	b.buf = append(b.buf, tmp...)
}

func (b *Buffer) PutFloat32(v float32) {
	b.off += 4

	tmp := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmp, math.Float32bits(v))
	b.buf = append(b.buf, tmp...)
}

func (b *Buffer) PutString(v string) {
	b.off += len(v) + 1

	for i := 0; i < len(v); i++ {
		b.PutByte(byte(v[i]))
	}

	b.PutByte(0)
}

func (b *Buffer) PutCoord16(v float32) {
	b.PutUint16(uint16(v * 8.0))
}

func (b *Buffer) PutCoord32(v float32) {
	b.PutFloat32(v)
}

func (b *Buffer) PutAngle8(v float32) {
	b.PutByte(byte(v / (360.0 / 256)))
}

func (b *Buffer) PutAngle16(v float32) {
	b.PutUint16(uint16(v / (360.0 / 65536)))
}

func (b *Buffer) PutAngle32(v float32) {
	b.PutFloat32(v)
}
