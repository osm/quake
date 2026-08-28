package state

import (
	"encoding/binary"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/internal/wire"
)

const entityFieldMask = protocol.URemove - 1

const (
	playerRecordWords = 12
	entityRecordSize  = 32
)

type PlayerRecord [playerRecordWords]uint32
type PlayerRecordBytes [playerRecordWords * 4]byte

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

func PlayerRecordBytesLE(r PlayerRecord) PlayerRecordBytes {
	var b PlayerRecordBytes

	for i := range r {
		binary.LittleEndian.PutUint32(b[i*4:(i+1)*4], r[i])
	}

	return b
}

func PlayerRecordFromBytesLE(b PlayerRecordBytes) PlayerRecord {
	var r PlayerRecord

	for i := range r {
		r[i] = binary.LittleEndian.Uint32(b[i*4 : (i+1)*4])
	}

	return r
}

func PlayerRecordUint16(b *PlayerRecordBytes, offset int) uint16 {
	return binary.LittleEndian.Uint16(b[offset : offset+2])
}

func SetPlayerRecordUint16(b *PlayerRecordBytes, offset int, value uint16) {
	binary.LittleEndian.PutUint16(b[offset:offset+2], value)
}

func EntityNumber(record EntityRecord) uint16 {
	return binary.LittleEndian.Uint16(record[wire.EntityNumberOffset:])
}

func SetEntityNumber(record *EntityRecord, number uint16) {
	binary.LittleEndian.PutUint16(record[wire.EntityNumberOffset:], number)
}

func EntityMask(record EntityRecord) uint16 {
	return binary.LittleEndian.Uint16(record[wire.EntityMaskOffset:])
}

func SetEntityMask(record *EntityRecord, mask uint16) {
	binary.LittleEndian.PutUint16(record[wire.EntityMaskOffset:], mask&entityFieldMask)
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
