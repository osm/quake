package wire

import "github.com/osm/quake/protocol"

const UntrackedRecordOffset = -1

type PacketEntityField struct {
	Mask         uint16
	RecordOffset int
	Size         int
}

var PacketEntityFields = [...]PacketEntityField{
	// Exact QuakeWorld wire order; neither mask bits nor history offsets
	// are sorted.
	{protocol.UModel, 4, 1},
	{protocol.UFrame, 5, 1},
	{protocol.UColorMap, 6, 1},
	{protocol.USkin, 7, 1},
	{protocol.UEffects, 8, 1},
	{protocol.UOrigin1, 12, 2},
	{protocol.UAngle1, 9, 1},
	{protocol.UOrigin2, 14, 2},
	{protocol.UAngle2, 10, 1},
	{protocol.UOrigin3, 16, 2},
	{protocol.UAngle3, 11, 1},
}

type PlayerCommandField struct {
	Mask         byte
	RecordOffset int
	Size         int
}

var PlayerCommandFields = [...]PlayerCommandField{
	// delta_usercmd wire order; angle 3 is consumed but not tracked.
	{protocol.CMAngle1, 14, 2},
	{protocol.CMAngle2, 12, 2},
	{protocol.CMAngle3, UntrackedRecordOffset, 2},
	{protocol.CMForward, 20, 2},
	{protocol.CMSide, 22, 2},
	{protocol.CMUp, 18, 2},
	{protocol.CMButtons, 17, 1},
	{protocol.CMImpulse, 16, 1},
}

type PlayerField struct {
	Mask         uint16
	RecordOffset int
	Size         int
}

var PlayerVelocityFields = [...]PlayerField{
	{protocol.PFVelocity1, 28, 2},
	{protocol.PFVelocity2, 30, 2},
	{protocol.PFVelocity3, 32, 2},
}

var PlayerByteFields = [...]PlayerField{
	{protocol.PFModel, 25, 1},
	{protocol.PFSkinNum, 26, 1},
	{protocol.PFEffects, 27, 1},
	{protocol.PFWeaponFrame, 34, 1},
}
