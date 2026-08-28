package wire

import "github.com/osm/quake/protocol"

const (
	UntrackedRecordOffset = -1

	EntityNumberOffset       = 0
	EntityMaskOffset         = 2
	EntityVisibleStateOffset = 4
	EntityModelOffset        = 4
	EntityFrameOffset        = 5
	EntityColorMapOffset     = 6
	EntitySkinNumOffset      = 7
	EntityEffectsOffset      = 8
	EntityAngleOffset        = 9
	EntityOriginOffset       = 12
	EntityVisibleStateEnd    = 18
	EntityOriginCarryOffset  = 18
	EntityAngleCarryOffset   = 24
	EntityCarryEnd           = 27

	PlayerOriginMaskOffset          = 0
	PlayerAngleMoveMaskOffset       = 1
	PlayerStateMaskOffset           = 2
	PlayerMotionMaskOffset          = 3
	PlayerOriginOffset              = 4
	PlayerFrameOffset               = 10
	PlayerIndexOffset               = 11
	PlayerAngleOffset               = 12
	PlayerImpulseOffset             = 16
	PlayerButtonsOffset             = 17
	PlayerRollOffset                = 18
	PlayerMoveOffset                = 20
	PlayerCommandMsecOffset         = 24
	PlayerModelOffset               = 25
	PlayerSkinNumOffset             = 26
	PlayerEffectsOffset             = 27
	PlayerVelocityOffset            = 28
	PlayerWeaponFrameOffset         = 34
	PlayerVisibleStateEnd           = PlayerWeaponFrameOffset + 1
	PlayerAngleAccumulatorOffset    = 36
	PlayerVelocityAccumulatorOffset = 40
	PlayerMsecOffset                = 46
)

const (
	PlayerFrameDelta        byte = 1 << 6
	PlayerOriginHistoryMask      = 0xff ^ PlayerFrameDelta
)

const (
	PlayerRollDeltaLo           = 1 << 0
	PlayerRollDeltaHi           = 1 << 1
	PlayerButtonsXOR       byte = 1 << 2
	PlayerImpulseSet            = 1 << 3
	PlayerCommandMsecDelta      = 1 << 4
	PlayerModelRemap            = 1 << 5
	PlayerSkinNumSet            = 1 << 6
	PlayerEffectsXOR            = 1 << 7

	PlayerStateHistoryMask = 0xff ^ PlayerImpulseSet ^ PlayerModelRemap ^ PlayerSkinNumSet
)

const (
	PlayerWeaponFrameDelta byte = 1 << 6
	PlayerDead                  = 1 << 7

	PlayerMotionHistoryMask = 0xff ^ PlayerWeaponFrameDelta
)

const PlayerMsecShortcut byte = 1 << 7

type PacketEntityField struct {
	Mask         uint16
	RecordOffset int
	Size         int
}

var PacketEntityFields = [...]PacketEntityField{
	// Exact QuakeWorld wire order; neither mask bits nor history offsets
	// are sorted.
	{protocol.UModel, EntityModelOffset, 1},
	{protocol.UFrame, EntityFrameOffset, 1},
	{protocol.UColorMap, EntityColorMapOffset, 1},
	{protocol.USkin, EntitySkinNumOffset, 1},
	{protocol.UEffects, EntityEffectsOffset, 1},
	{protocol.UOrigin1, EntityOriginOffset, 2},
	{protocol.UAngle1, EntityAngleOffset, 1},
	{protocol.UOrigin2, EntityOriginOffset + 2, 2},
	{protocol.UAngle2, EntityAngleOffset + 1, 1},
	{protocol.UOrigin3, EntityOriginOffset + 4, 2},
	{protocol.UAngle3, EntityAngleOffset + 2, 1},
}

type PlayerCommandField struct {
	Mask         byte
	RecordOffset int
	Size         int
}

var PlayerCommandFields = [...]PlayerCommandField{
	// delta_usercmd wire order; angle 3 is consumed but not tracked.
	{protocol.CMAngle1, PlayerAngleOffset + 2, 2},
	{protocol.CMAngle2, PlayerAngleOffset, 2},
	{protocol.CMAngle3, UntrackedRecordOffset, 2},
	{protocol.CMForward, PlayerMoveOffset, 2},
	{protocol.CMSide, PlayerMoveOffset + 2, 2},
	{protocol.CMUp, PlayerRollOffset, 2},
	{protocol.CMButtons, PlayerButtonsOffset, 1},
	{protocol.CMImpulse, PlayerImpulseOffset, 1},
}

type PlayerField struct {
	Mask         uint16
	RecordOffset int
	Size         int
}

var PlayerVelocityFields = [...]PlayerField{
	{protocol.PFVelocity1, PlayerVelocityOffset, 2},
	{protocol.PFVelocity2, PlayerVelocityOffset + 2, 2},
	{protocol.PFVelocity3, PlayerVelocityOffset + 4, 2},
}

var PlayerByteFields = [...]PlayerField{
	{protocol.PFModel, PlayerModelOffset, 1},
	{protocol.PFSkinNum, PlayerSkinNumOffset, 1},
	{protocol.PFEffects, PlayerEffectsOffset, 1},
	{protocol.PFWeaponFrame, PlayerWeaponFrameOffset, 1},
}
