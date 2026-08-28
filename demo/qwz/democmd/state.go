package democmd

import (
	"github.com/osm/quake/demo/qwd"
	"github.com/osm/quake/qizmo/freq"
)

const (
	degreesPerTurn = 360.0
	unitsPerTurn   = 1 << 16
	initialImpulse = 8

	PayloadSize = qwd.CmdPayloadSize
)

func unpackAngle(angle int16) float32 {
	return float32(float64(angle) * (degreesPerTurn / unitsPerTurn))
}

type State struct {
	commandMask uint32
	angles      [3]int16
	angleDeltas [3]int16
	movement    [3]int16
	buttons     byte
	impulse     byte
	msec        byte
}

func NewState() *State {
	return &State{impulse: initialImpulse}
}

func (s *State) Msec() byte {
	return s.msec
}

type wordField struct {
	lowMask  uint32
	highMask uint32
	lowRow   uint32
	highRow  uint32
}

// The field order matches the QWD usercmd: pitch, yaw, roll, forward, side, up.
var wordFields = [...]wordField{
	{0x0001, 0x0002, freq.CmdPitchDeltaLo, freq.CmdPitchDeltaHi},
	{0x0004, 0x0008, freq.CmdYawDeltaLo, freq.CmdYawDeltaHi},
	{0x0010, 0x0020, freq.CmdRollDeltaLo, freq.CmdRollDeltaHi},
	{0x0040, 0x0080, freq.CmdForwardDeltaLo, freq.CmdForwardDeltaHi},
	{0x0100, 0x0200, freq.CmdSideDeltaLo, freq.CmdSideDeltaHi},
	{0x0400, 0x0800, freq.CmdUpmoveDeltaLo, freq.CmdUpmoveDeltaHi},
}
