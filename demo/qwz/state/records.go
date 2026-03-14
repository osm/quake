package state

import (
	"encoding/binary"
)

type PlayerRecord [12]uint32

type PlayerHistory struct {
	Seq   uint32
	Valid bool
	Recs  []PlayerRecord
}

type EntityRecord [32]byte

type EntityHistory struct {
	Seq     uint32
	Valid   bool
	Ents    map[uint16]EntityRecord
	Ordered []EntityRecord
}

func PlayerRecordByte(r PlayerRecord, off int) byte {
	i := off / 4
	shift := uint((off % 4) * 8)
	return byte((r[i] >> shift) & 0xff)
}

func SetPlayerRecordByte(r *PlayerRecord, off int, v byte) {
	i := off / 4
	shift := uint((off % 4) * 8)
	mask := uint32(0xff) << shift
	r[i] = (r[i] & ^mask) | (uint32(v) << shift)
}

func PlayerRecordBytesLE(r PlayerRecord) [48]byte {
	var b [48]byte

	for i := 0; i < 12; i++ {
		binary.LittleEndian.PutUint32(b[i*4:(i+1)*4], r[i])
	}

	return b
}

func PlayerRecordFromBytesLE(b [48]byte) PlayerRecord {
	var r PlayerRecord

	for i := 0; i < 12; i++ {
		r[i] = binary.LittleEndian.Uint32(b[i*4 : (i+1)*4])
	}

	return r
}

func GetU16LE(b *[48]byte, off int) uint16 {
	return binary.LittleEndian.Uint16(b[off : off+2])
}

func SetU16LE(b *[48]byte, off int, v uint16) {
	binary.LittleEndian.PutUint16(b[off:off+2], v)
}

func EntityNumber(r EntityRecord) uint16 {
	return binary.LittleEndian.Uint16(r[0:2])
}

func SetEntityNumber(r *EntityRecord, n uint16) {
	binary.LittleEndian.PutUint16(r[0:2], n)
}

func EntityRecordByte(r EntityRecord, off int) byte {
	return r[off]
}

func SetEntityRecordByte(r *EntityRecord, off int, v byte) {
	r[off] = v
}

func EntityRecordU16(r EntityRecord, off int) uint16 {
	return binary.LittleEndian.Uint16(r[off : off+2])
}

func SetEntityRecordU16(r *EntityRecord, off int, v uint16) {
	binary.LittleEndian.PutUint16(r[off:off+2], v)
}

func AddEntityRecordI16(r *EntityRecord, off int, d int16) {
	v := int16(EntityRecordU16(*r, off)) + d
	SetEntityRecordU16(r, off, uint16(v))
}
