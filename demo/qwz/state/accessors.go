package state

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

func PlayerFreqByte(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x19)
}

func SetPlayerFreqByte(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x19, v)
}

func PlayerEffectsByte(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x1a)
}

func SetPlayerEffectsByte(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x1a, v)
}

func PlayerSkinByte(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x1b)
}

func SetPlayerSkinByte(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x1b, v)
}

func PlayerOriginZByte2(r PlayerRecord) byte {
	return PlayerRecordByte(r, 0x0a)
}
func SetPlayerOriginZByte2(r *PlayerRecord, v byte) {
	SetPlayerRecordByte(r, 0x0a, v)
}
