package democmd

import (
	"encoding/binary"
	"math"

	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/rangedec"
)

func Decode(rd *rangedec.Decoder, ft *freq.Tables, st *State) ([PayloadSize]byte, error) {
	var out [PayloadSize]byte

	loSym, err := rd.DecodeSymbol(ft.CumulativeRow(freq.CmdMaskLo), freq.Symbols)
	if err != nil {
		return out, err
	}
	hiSym, err := rd.DecodeSymbol(ft.CumulativeRow(freq.CmdMaskHi), freq.Symbols)
	if err != nil {
		return out, err
	}

	commandMask := (st.commandMask & qizmoprotocol.CMPredictionMask) ^ loSym ^ hiSym<<8

	angles := st.angles
	angleDeltas := st.angleDeltas
	movement := st.movement
	for i, field := range wordFields[:len(angles)] {
		angleDeltas[i], err = decodeWordDelta(rd, ft, commandMask, field, angleDeltas[i])
		if err != nil {
			return out, err
		}
		angles[i] = packed.AddWrap16(angles[i], angleDeltas[i])
	}
	for i, field := range wordFields[len(angles):] {
		movement[i], err = decodeWordDelta(rd, ft, commandMask, field, movement[i])
		if err != nil {
			return out, err
		}
	}
	if commandMask&qizmoprotocol.CMButtons != 0 {
		v, err := rd.DecodeSymbol(ft.CumulativeRow(freq.CmdButtonsXOR), freq.Symbols)
		if err != nil {
			return out, err
		}
		st.buttons ^= byte(v)
	}
	if commandMask&qizmoprotocol.CMMsec != 0 {
		v, err := rd.DecodeSymbol(ft.CumulativeRow(freq.CmdMsecDelta), freq.Symbols)
		if err != nil {
			return out, err
		}
		st.msec = byte(int8(st.msec) + int8(byte(v)))
	}
	if commandMask&qizmoprotocol.CMImpulse != 0 {
		v, err := rd.DecodeSymbol(ft.CumulativeRow(freq.CmdImpulseSet), freq.Symbols)
		if err != nil {
			return out, err
		}
		st.impulse = byte(v)
	}

	out[msecOffset] = st.msec
	for i, angle := range angles {
		value := unpackAngle(angle)
		bits := math.Float32bits(value)
		binary.LittleEndian.PutUint32(out[angleOffset+i*4:], bits)
		binary.LittleEndian.PutUint32(out[angleCopyOffset+i*4:], bits)
	}
	for i, value := range movement {
		binary.LittleEndian.PutUint16(out[movementOffset+i*2:], uint16(value))
	}
	out[buttonsOffset] = st.buttons
	out[impulseOffset] = st.impulse

	st.commandMask = commandMask
	st.angles = angles
	st.angleDeltas = angleDeltas
	st.movement = movement
	return out, nil
}

func decodeWordDelta(
	rd *rangedec.Decoder,
	ft *freq.Tables,
	mask uint32,
	field wordField,
	value int16,
) (int16, error) {
	if mask&field.lowMask != 0 {
		symbol, err := rd.DecodeSymbol(ft.CumulativeRow(field.lowRow), freq.Symbols)
		if err != nil {
			return 0, err
		}
		value = packed.AddWrap16(value, int16(int8(byte(symbol))))
	}
	if mask&field.highMask != 0 {
		symbol, err := rd.DecodeSymbol(ft.CumulativeRow(field.highRow), freq.Symbols)
		if err != nil {
			return 0, err
		}
		value = packed.AddWrap16(value, int16(uint16(byte(symbol))<<8))
	}
	return value, nil
}
