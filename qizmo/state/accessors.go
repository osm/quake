package state

const entityOriginOffset = 12

func PlayerOriginMask(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x00)
}

func SetPlayerOriginMask(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x00, v)
}

func PlayerStateMask(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x02)
}

func SetPlayerStateMask(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x02, v)
}

func PlayerMotionMask(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x03)
}

func SetPlayerMotionMask(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x03, v)
}

func PlayerModel(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x19)
}

func SetPlayerModel(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x19, v)
}

func PlayerSkinNum(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x1a)
}

func SetPlayerSkinNum(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x1a, v)
}

func PlayerEffects(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x1b)
}

func SetPlayerEffects(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x1b, v)
}

func PlayerFrame(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x0a)
}

func SetPlayerFrame(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x0a, v)
}

func PlayerIndex(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x0b)
}

func SetPlayerIndex(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x0b, v)
}

func PlayerWeaponFrame(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x22)
}

func EntityOrigin(r EntityRecord) (origin [3]uint16) {
	for axis := range origin {
		origin[axis] = EntityRecordUint16(r, entityOriginOffset+axis*2)
	}
	return origin
}

func SetEntityOrigin(r *EntityRecord, origin [3]uint16) {
	for axis, coordinate := range origin {
		SetEntityRecordUint16(r, entityOriginOffset+axis*2, coordinate)
	}
}
