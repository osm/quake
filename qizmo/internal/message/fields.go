package message

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangedec"
)

const maxPlayers = protocol.QWMaxClients

const (
	setInfoKeyOffset           = 1
	updateUserInfoUserIDOffset = 1
	updateUserInfoUserIDSize   = 4
	updateUserInfoStringOffset = updateUserInfoUserIDOffset + updateUserInfoUserIDSize
)

const (
	printDictionaryStart = 0x120
	chatDictionaryStart  = 0x140
)

const (
	coordinateTripletSize = 3 * 2
	nailProjectileSize    = 6
	soundEntityShift      = 3
	soundEntityMask       = (1 << 9) - 1
)

func endCString(data []byte, offset int) (int, bool) {
	for offset < len(data) {
		offset++
		if data[offset-1] == 0 {
			return offset, true
		}
	}
	return offset, false
}

type wordDeltaRows struct {
	low  uint32
	high uint32
}

func decodeMaskedWordDelta(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	mask, lowMask, highMask uint16,
	rows wordDeltaRows,
) (int16, error) {
	var delta int16
	if mask&lowMask != 0 {
		value, err := rd.DecodeFreqByte(ft, rows.low)
		if err != nil {
			return 0, err
		}
		delta += int16(int8(value))
	}
	if mask&highMask != 0 {
		value, err := rd.DecodeFreqByte(ft, rows.high)
		if err != nil {
			return 0, err
		}
		delta += int16(uint16(value) << 8)
	}
	return delta, nil
}

func decodeCoordinateTriplet(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	mask uint16,
	rows [3]wordDeltaRows,
	coordinates *[3]uint16,
) error {
	for axis, axisRows := range rows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(rd, ft, mask, lowMask, lowMask<<1, axisRows)
		if err != nil {
			return err
		}
		coordinates[axis] = uint16(int16(coordinates[axis]) + delta)
	}
	return nil
}

var (
	playerOriginDeltaRows = [3]wordDeltaRows{
		{freq.SVCPlayerInfoOriginXLoDelta, freq.SVCPlayerInfoOriginXHiDelta},
		{freq.SVCPlayerInfoOriginYLoDelta, freq.SVCPlayerInfoOriginYHiDelta},
		{freq.SVCPlayerInfoOriginZLoDelta, freq.SVCPlayerInfoOriginZHiDelta},
	}
	playerAngleMoveDeltaRows = [4]wordDeltaRows{
		{freq.SVCPlayerInfoPitchLoDelta, freq.SVCPlayerInfoPitchHiDelta},
		{freq.SVCPlayerInfoYawLoDelta, freq.SVCPlayerInfoYawHiDelta},
		{freq.SVCPlayerInfoForwardLoDelta, freq.SVCPlayerInfoForwardHiDelta},
		{freq.SVCPlayerInfoSideLoDelta, freq.SVCPlayerInfoSideHiDelta},
	}
	playerVelocityDeltaRows = [3]wordDeltaRows{
		{freq.SVCPlayerInfoVelXLoDelta, freq.SVCPlayerInfoVelXHiDelta},
		{freq.SVCPlayerInfoVelYLoDelta, freq.SVCPlayerInfoVelYHiDelta},
		{freq.SVCPlayerInfoVelZLoDelta, freq.SVCPlayerInfoVelZHiDelta},
	}
	playerRollDeltaRows = wordDeltaRows{
		freq.SVCPlayerInfoRollLoDelta,
		freq.SVCPlayerInfoRollHiDelta,
	}
	coordinateDeltaRows = [3]wordDeltaRows{
		{freq.CoordDeltaXLo, freq.CoordDeltaXHi},
		{freq.CoordDeltaYLo, freq.CoordDeltaYHi},
		{freq.CoordDeltaZLo, freq.CoordDeltaZHi},
	}
	tempEntityCoordinateDeltaRows = [3]wordDeltaRows{
		{freq.SVCTEntCoordXLoDelta, freq.SVCTEntCoordXHiDelta},
		{freq.SVCTEntCoordYLoDelta, freq.SVCTEntCoordYHiDelta},
		{freq.SVCTEntCoordZLoDelta, freq.SVCTEntCoordZHiDelta},
	}
	damageCoordinateDeltaRows = [3]wordDeltaRows{
		{freq.SVCDamageFromXLo, freq.SVCDamageFromXHi},
		{freq.SVCDamageFromYLo, freq.SVCDamageFromYHi},
		{freq.SVCDamageFromZLo, freq.SVCDamageFromZHi},
	}
	packetEntityPositionDeltaRows = [3]wordDeltaRows{
		{freq.SVCPacketEntityPosXLoDelta, freq.SVCPacketEntityPosXHiDelta},
		{freq.SVCPacketEntityPosYLoDelta, freq.SVCPacketEntityPosYHiDelta},
		{freq.SVCPacketEntityPosZLoDelta, freq.SVCPacketEntityPosZHiDelta},
	}
	primaryPlayerMaskXORRows = [3]uint32{
		freq.SVCPlayerInfoOriginMaskXOR,
		freq.SVCPlayerInfoStateMaskXOR,
		freq.SVCPlayerInfoMotionMaskXOR,
	}
	playerMaskDeltaRows = [4]uint32{
		freq.SVCPlayerInfoOriginMaskDelta,
		freq.SVCPlayerInfoAngleMoveMaskDelta,
		freq.SVCPlayerInfoStateMaskDelta,
		freq.SVCPlayerInfoVelocityMaskDelta,
	}
	nailProjectileDeltaRows = [nailProjectileSize]uint32{
		freq.SVCNailsProjectileByte0,
		freq.SVCNailsProjectileByte1,
		freq.SVCNailsProjectileByte2,
		freq.SVCNailsProjectileByte3,
		freq.SVCNailsProjectileByte4,
		freq.SVCNailsProjectileByte5,
	}
)

type packetEntityFieldKind byte

const (
	packetEntityFieldDelta packetEntityFieldKind = iota
	packetEntityFieldModel
	packetEntityFieldXOR
)

const (
	packetEntityModelFlag byte = 1 << iota
	packetEntityFrameFlag
	packetEntityColorMapFlag
	packetEntitySkinNumFlag
	packetEntityEffectsFlag
	packetEntityAngle1Flag
	packetEntityAngle2Flag
	packetEntityAngle3Flag
)

type packetEntityFieldSpec struct {
	mask   byte
	offset int
	row    uint32
	kind   packetEntityFieldKind
}

var packetEntityFields = [...]packetEntityFieldSpec{
	{packetEntityModelFlag, 4, freq.SVCPacketEntityModelRemapIndex, packetEntityFieldModel},
	{packetEntityFrameFlag, 5, freq.SVCPacketEntityFrameDelta, packetEntityFieldDelta},
	{packetEntityColorMapFlag, 6, freq.SVCPacketEntityColorMapDelta, packetEntityFieldDelta},
	{packetEntitySkinNumFlag, 7, freq.SVCPacketEntitySkinDelta, packetEntityFieldDelta},
	{packetEntityEffectsFlag, 8, freq.SVCPacketEntityEffectsXOR, packetEntityFieldXOR},
	{packetEntityAngle1Flag, 9, freq.SVCPacketEntityAngle1Delta, packetEntityFieldDelta},
	{packetEntityAngle2Flag, 10, freq.SVCPacketEntityAngle2Delta, packetEntityFieldDelta},
	{packetEntityAngle3Flag, 11, freq.SVCPacketEntityAngle3Delta, packetEntityFieldDelta},
}
