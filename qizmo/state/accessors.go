package state

import "github.com/osm/quake/qizmo/internal/wire"

const (
	playerOriginWord              = wire.PlayerOriginOffset / 4
	playerVelocityWord            = wire.PlayerVelocityOffset / 4
	playerVelocityAccumulatorWord = wire.PlayerVelocityAccumulatorOffset / 4
)

func PlayerOriginMask(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerOriginMaskOffset)
}

func SetPlayerOriginMask(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerOriginMaskOffset, v)
}

func PlayerStateMask(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerStateMaskOffset)
}

func SetPlayerStateMask(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerStateMaskOffset, v)
}

func PlayerMotionMask(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerMotionMaskOffset)
}

func SetPlayerMotionMask(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerMotionMaskOffset, v)
}

func PlayerModel(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerModelOffset)
}

func SetPlayerModel(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerModelOffset, v)
}

func PlayerSkinNum(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerSkinNumOffset)
}

func SetPlayerSkinNum(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerSkinNumOffset, v)
}

func PlayerEffects(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerEffectsOffset)
}

func SetPlayerEffects(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerEffectsOffset, v)
}

func PlayerFrame(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerFrameOffset)
}

func SetPlayerFrame(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerFrameOffset, v)
}

func PlayerIndex(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerIndexOffset)
}

func SetPlayerIndex(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerIndexOffset, v)
}

func PlayerWeaponFrame(r PlayerRecord) byte {
	return PlayerRecordByte(r, wire.PlayerWeaponFrameOffset)
}

func SetPlayerWeaponFrame(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, wire.PlayerWeaponFrameOffset, v)
}

func EntityOrigin(r EntityRecord) (origin [3]uint16) {
	for axis := range origin {
		origin[axis] = EntityRecordUint16(r, wire.EntityOriginOffset+axis*2)
	}
	return origin
}

func SetEntityOrigin(r *EntityRecord, origin [3]uint16) {
	for axis, coordinate := range origin {
		SetEntityRecordUint16(r, wire.EntityOriginOffset+axis*2, coordinate)
	}
}

func PlayerOrigin(r PlayerRecord) [3]uint16 {
	return [3]uint16{
		uint16(r[playerOriginWord]),
		uint16(r[playerOriginWord] >> 16),
		uint16(r[playerOriginWord+1]),
	}
}

func SetPlayerOrigin(r *PlayerRecord, origin [3]uint16) {
	r[playerOriginWord] = uint32(origin[0]) | uint32(origin[1])<<16
	r[playerOriginWord+1] = r[playerOriginWord+1]&0xffff0000 | uint32(origin[2])
}

func PlayerVelocity(r PlayerRecord) [3]int16 {
	return playerVector(r, playerVelocityWord)
}

func SetPlayerVelocity(r *PlayerRecord, velocity [3]int16) {
	setPlayerVector(r, playerVelocityWord, velocity)
}

func PlayerVelocityAccumulator(r PlayerRecord) [3]int16 {
	return playerVector(r, playerVelocityAccumulatorWord)
}

func SetPlayerVelocityAccumulator(r *PlayerRecord, accumulator [3]int16) {
	setPlayerVector(r, playerVelocityAccumulatorWord, accumulator)
}

func playerVector(r PlayerRecord, word int) [3]int16 {
	return [3]int16{
		int16(r[word]),
		int16(r[word] >> 16),
		int16(r[word+1]),
	}
}

func setPlayerVector(r *PlayerRecord, word int, vector [3]int16) {
	r[word] = uint32(uint16(vector[0])) | uint32(uint16(vector[1]))<<16
	r[word+1] = r[word+1]&0xffff0000 | uint32(uint16(vector[2]))
}
