package state

import "encoding/binary"

const (
	playerRecordWords = 12
	entityRecordSize  = 32
)

type PlayerRecord [playerRecordWords]uint32

type playerHistory struct {
	sequence uint32
	valid    bool
	players  []PlayerRecord
}

type EntityRecord [entityRecordSize]byte

type entityHistory struct {
	sequence uint32
	valid    bool
	entities map[uint16]EntityRecord
	ordered  []EntityRecord
}

func PlayerRecordByte(record PlayerRecord, offset int) byte {
	word := offset / 4
	shift := uint((offset % 4) * 8)
	return byte((record[word] >> shift) & 0xff)
}

func SetPlayerRecordByte(record *PlayerRecord, offset int, value byte) {
	word := offset / 4
	shift := uint((offset % 4) * 8)
	mask := uint32(0xff) << shift
	record[word] = (record[word] & ^mask) | (uint32(value) << shift)
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

func PlayerRecordUint16(b *[48]byte, offset int) uint16 {
	return binary.LittleEndian.Uint16(b[offset : offset+2])
}

func SetPlayerRecordUint16(b *[48]byte, offset int, value uint16) {
	binary.LittleEndian.PutUint16(b[offset:offset+2], value)
}

func EntityNumber(record EntityRecord) uint16 {
	return binary.LittleEndian.Uint16(record[0:2])
}

func SetEntityNumber(record *EntityRecord, number uint16) {
	binary.LittleEndian.PutUint16(record[0:2], number)
}

func EntityRecordByte(record EntityRecord, offset int) byte {
	return record[offset]
}

func SetEntityRecordByte(record *EntityRecord, offset int, value byte) {
	record[offset] = value
}

func EntityRecordUint16(record EntityRecord, offset int) uint16 {
	return binary.LittleEndian.Uint16(record[offset : offset+2])
}

func SetEntityRecordUint16(record *EntityRecord, offset int, value uint16) {
	binary.LittleEndian.PutUint16(record[offset:offset+2], value)
}

func AddEntityRecordInt16(record *EntityRecord, offset int, delta int16) {
	value := int16(EntityRecordUint16(*record, offset)) + delta
	SetEntityRecordUint16(record, offset, uint16(value))
}
