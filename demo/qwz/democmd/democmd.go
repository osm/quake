package democmd

import (
	"encoding/binary"
	"math"

	"github.com/osm/quake/demo/qwz/freq"
	"github.com/osm/quake/demo/qwz/packed"
	"github.com/osm/quake/demo/qwz/rangedec"
)

type DemoCmd struct {
	CommandMask uint32

	Angle0      int16
	Angle1      int16
	Angle2      int16
	Angle0Delta int16
	Angle1Delta int16
	Angle2Delta int16

	ForwardSide uint32
	Up          int16
	Buttons     byte
	Impulse     byte
	Msec        byte
}

func Decode(rd *rangedec.Decoder, ft *freq.Tables, st *DemoCmd) ([0x24]byte, error) {
	var out [0x24]byte

	maskLoCum := ft.CumulativeRow(freq.CmdMaskLo)
	maskHiCum := ft.CumulativeRow(freq.CmdMaskHi)
	pitchDeltaLoCum := ft.CumulativeRow(freq.CmdPitchDeltaLo)
	pitchDeltaHiCum := ft.CumulativeRow(freq.CmdPitchDeltaHi)
	yawDeltaLoCum := ft.CumulativeRow(freq.CmdYawDeltaLo)
	yawDeltaHiCum := ft.CumulativeRow(freq.CmdYawDeltaHi)
	rollDeltaLoCum := ft.CumulativeRow(freq.CmdRollDeltaLo)
	rollDeltaHiCum := ft.CumulativeRow(freq.CmdRollDeltaHi)
	forwardDeltaHiLoCum := ft.CumulativeRow(freq.CmdForwardHiLoDelta)
	forwardDeltaHiHiCum := ft.CumulativeRow(freq.CmdForwardHiHiDelta)
	forwardDeltaLoLoCum := ft.CumulativeRow(freq.CmdForwardLoLoDelta)
	forwardDeltaLoHiCum := ft.CumulativeRow(freq.CmdForwardLoHiDelta)
	upmoveDeltaLoCum := ft.CumulativeRow(freq.CmdUpmoveDeltaLo)
	upmoveDeltaHiCum := ft.CumulativeRow(freq.CmdUpmoveDeltaHi)
	buttonsXorCum := ft.CumulativeRow(freq.CmdButtonsXor)
	impulseSetCum := ft.CumulativeRow(freq.CmdImpulseSet)
	msecDeltaCum := ft.CumulativeRow(freq.CmdMsecDelta)

	loSym, err := rd.DecodeSymbol(maskLoCum, 0x100)
	if err != nil {
		return out, err
	}
	hiSym, err := rd.DecodeSymbol(maskHiCum, 0x100)
	if err != nil {
		return out, err
	}

	lo8 := uint32(loSym) & 0xff
	hi8 := uint32(hiSym) & 0xff

	prevMask := uint32(st.CommandMask)
	partialMask := (prevMask & 0x203f) ^ lo8
	commandMask := partialMask ^ (hi8 << 8)
	lowMaskSigned := int8(byte(partialMask & 0xff))

	if (commandMask & 0x0001) != 0 {
		v, err := rd.DecodeSymbol(pitchDeltaLoCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Angle0Delta = packed.AddWrap16(st.Angle0Delta, int16(int8(byte(v))))
	}
	if (commandMask & 0x0002) != 0 {
		v, err := rd.DecodeSymbol(pitchDeltaHiCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Angle0Delta = packed.AddWrap16(st.Angle0Delta, int16(uint16(byte(v))<<8))
	}
	st.Angle0 = packed.AddWrap16(st.Angle0, st.Angle0Delta)

	if (commandMask & 0x0004) != 0 {
		v, err := rd.DecodeSymbol(yawDeltaLoCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Angle1Delta = packed.AddWrap16(st.Angle1Delta, int16(int8(byte(v))))
	}
	if (commandMask & 0x0008) != 0 {
		v, err := rd.DecodeSymbol(yawDeltaHiCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Angle1Delta = packed.AddWrap16(st.Angle1Delta, int16(uint16(byte(v))<<8))
	}
	st.Angle1 = packed.AddWrap16(st.Angle1, st.Angle1Delta)

	if (commandMask & 0x0010) != 0 {
		v, err := rd.DecodeSymbol(rollDeltaLoCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Angle2Delta = packed.AddWrap16(st.Angle2Delta, int16(int8(byte(v))))
	}
	if (commandMask & 0x0020) != 0 {
		v, err := rd.DecodeSymbol(rollDeltaHiCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Angle2Delta = packed.AddWrap16(st.Angle2Delta, int16(uint16(byte(v))<<8))
	}
	st.Angle2 = packed.AddWrap16(st.Angle2, st.Angle2Delta)

	if (commandMask & 0x0040) != 0 {
		v, err := rd.DecodeSymbol(forwardDeltaHiLoCum, 0x100)
		if err != nil {
			return out, err
		}
		hi16 := int16(uint16(st.ForwardSide >> 16))
		hi16 += int16(int8(byte(v)))
		st.ForwardSide = (uint32(uint16(hi16)) << 16) | (st.ForwardSide & 0xffff)
	}
	if lowMaskSigned < 0 {
		v, err := rd.DecodeSymbol(forwardDeltaHiHiCum, 0x100)
		if err != nil {
			return out, err
		}
		hi16 := int16(uint16(st.ForwardSide >> 16))
		hi16 += int16(uint16(byte(v)) << 8)
		st.ForwardSide = (uint32(uint16(hi16)) << 16) | (st.ForwardSide & 0xffff)
	}

	moveFSBeforeUpdate := st.ForwardSide
	if (commandMask & 0x0100) != 0 {
		v, err := rd.DecodeSymbol(forwardDeltaLoLoCum, 0x100)
		if err != nil {
			return out, err
		}
		lo16 := int16(uint16(st.ForwardSide & 0xffff))
		lo16 += int16(int8(byte(v)))
		st.ForwardSide = (st.ForwardSide & 0xffff0000) | uint32(uint16(lo16))
	}
	if (commandMask & 0x0200) != 0 {
		v, err := rd.DecodeSymbol(forwardDeltaLoHiCum, 0x100)
		if err != nil {
			return out, err
		}
		lo16 := int16(uint16(st.ForwardSide & 0xffff))
		lo16 += int16(uint16(byte(v)) << 8)
		st.ForwardSide = (st.ForwardSide & 0xffff0000) | uint32(uint16(lo16))
	}
	if (commandMask & 0x0400) != 0 {
		v, err := rd.DecodeSymbol(upmoveDeltaLoCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Up = packed.AddWrap16(st.Up, int16(int8(byte(v))))
	}
	if (commandMask & 0x0800) != 0 {
		v, err := rd.DecodeSymbol(upmoveDeltaHiCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Up = packed.AddWrap16(st.Up, int16(uint16(byte(v))<<8))
	}
	if (commandMask & 0x1000) != 0 {
		v, err := rd.DecodeSymbol(buttonsXorCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Buttons ^= byte(v)
	}
	if (commandMask & 0x2000) != 0 {
		v, err := rd.DecodeSymbol(msecDeltaCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Msec = byte(int8(st.Msec) + int8(byte(v)))
	}
	if (commandMask & 0x4000) != 0 {
		v, err := rd.DecodeSymbol(impulseSetCum, 0x100)
		if err != nil {
			return out, err
		}
		st.Impulse = byte(v)
	}

	out[0x00] = st.Msec
	const angleScale = 360.0 / 65536.0
	f0 := float32(float64(st.Angle0) * angleScale)
	f1 := float32(float64(st.Angle1) * angleScale)
	f2 := float32(float64(st.Angle2) * angleScale)

	binary.LittleEndian.PutUint32(out[0x04:0x08], math.Float32bits(f0))
	binary.LittleEndian.PutUint32(out[0x08:0x0c], math.Float32bits(f1))
	binary.LittleEndian.PutUint32(out[0x0c:0x10], math.Float32bits(f2))

	out[0x10] = byte((moveFSBeforeUpdate >> 16) & 0xff)
	out[0x11] = byte((moveFSBeforeUpdate >> 24) & 0xff)
	out[0x12] = byte(st.ForwardSide & 0xff)
	out[0x13] = byte((st.ForwardSide >> 8) & 0xff)

	um := uint16(int16(st.Up))
	out[0x14] = byte(um & 0xff)
	out[0x15] = byte((um >> 8) & 0xff)
	out[0x16] = st.Buttons
	out[0x17] = st.Impulse

	binary.LittleEndian.PutUint32(out[0x18:0x1c], math.Float32bits(f0))
	binary.LittleEndian.PutUint32(out[0x1c:0x20], math.Float32bits(f1))
	binary.LittleEndian.PutUint32(out[0x20:0x24], math.Float32bits(f2))

	st.CommandMask = commandMask
	return out, nil
}
