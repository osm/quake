package democmd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/rangeenc"
)

func Encode(enc *rangeenc.Encoder, ft *freq.Tables, st *State, payload [PayloadSize]byte) error {
	for _, padding := range payload[paddingOffset:angleOffset] {
		if padding != 0 {
			return fmt.Errorf("non-zero usercmd alignment padding")
		}
	}
	if !bytes.Equal(payload[angleOffset:movementOffset], payload[angleCopyOffset:PayloadSize]) {
		return fmt.Errorf("usercmd angle copies differ")
	}

	angles := [3]int16{}
	for i := range angles {
		bits := binary.LittleEndian.Uint32(payload[angleOffset+i*4:])
		angle, ok := packedAngle(bits)
		if !ok {
			return fmt.Errorf("angle %d is not QWZ-representable: %v", i, math.Float32frombits(bits))
		}
		angles[i] = angle
	}

	previousAngles := st.angles
	previousAngleDeltas := st.angleDeltas
	var angleDeltas [3]int16
	var changes [6]packed.WordDelta
	for i, angle := range angles {
		angleDeltas[i] = int16(uint16(angle) - uint16(previousAngles[i]))
		changes[i] = packed.SplitWordDelta(angleDeltas[i], previousAngleDeltas[i])
	}

	previousMovement := st.movement
	var movement [3]int16
	for i := range movement {
		movement[i] = int16(binary.LittleEndian.Uint16(payload[movementOffset+i*2:]))
		changes[len(angles)+i] = packed.SplitWordDelta(movement[i], previousMovement[i])
	}

	buttonsXOR := payload[buttonsOffset] ^ st.buttons
	msecDelta := payload[msecOffset] - st.msec
	if payload[impulseOffset] == 0 && st.impulse != 0 {
		return fmt.Errorf("zero impulse cannot replace retained impulse %d", st.impulse)
	}
	// Qizmo transmits every non-zero impulse, even when it is unchanged from
	// the prior command. This is deliberately not a conventional set-on-change
	// delta: reproducing that redundant symbol is required for its exact stream.
	setImpulse := payload[impulseOffset] != 0

	var commandMask uint32
	for i, change := range changes {
		if change.Low != 0 {
			commandMask |= wordFields[i].lowMask
		}
		if change.High != 0 {
			commandMask |= wordFields[i].highMask
		}
	}
	if buttonsXOR != 0 {
		commandMask |= qizmoprotocol.CMButtons
	}
	if msecDelta != 0 {
		commandMask |= qizmoprotocol.CMMsec
	}
	if setImpulse {
		commandMask |= qizmoprotocol.CMImpulse
	}

	maskDelta := commandMask ^ (st.commandMask & qizmoprotocol.CMPredictionMask)
	maskLo := byte(maskDelta)
	maskHi := byte(maskDelta >> 8)

	if err := enc.EncodeSymbol(ft.CumulativeRow(freq.CmdMaskLo), uint32(maskLo)); err != nil {
		return err
	}
	if err := enc.EncodeSymbol(ft.CumulativeRow(freq.CmdMaskHi), uint32(maskHi)); err != nil {
		return err
	}

	for i, change := range changes {
		field := wordFields[i]
		if change.Low != 0 {
			if err := enc.EncodeSymbol(ft.CumulativeRow(field.lowRow), uint32(change.Low)); err != nil {
				return err
			}
		}
		if change.High != 0 {
			if err := enc.EncodeSymbol(ft.CumulativeRow(field.highRow), uint32(change.High)); err != nil {
				return err
			}
		}
	}
	if buttonsXOR != 0 {
		if err := enc.EncodeSymbol(ft.CumulativeRow(freq.CmdButtonsXOR), uint32(buttonsXOR)); err != nil {
			return err
		}
	}
	if msecDelta != 0 {
		if err := enc.EncodeSymbol(ft.CumulativeRow(freq.CmdMsecDelta), uint32(msecDelta)); err != nil {
			return err
		}
	}
	if setImpulse {
		if err := enc.EncodeSymbol(ft.CumulativeRow(freq.CmdImpulseSet), uint32(payload[impulseOffset])); err != nil {
			return err
		}
	}

	st.commandMask = commandMask
	st.angles = angles
	st.angleDeltas = angleDeltas
	st.movement = movement
	st.buttons = payload[buttonsOffset]
	st.impulse = payload[impulseOffset]
	st.msec = payload[msecOffset]
	return nil
}

func packedAngle(bits uint32) (int16, bool) {
	f := math.Float32frombits(bits)
	if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
		return 0, false
	}
	if f < -degreesPerTurn/2 || f >= degreesPerTurn/2 {
		return 0, false
	}

	estimate := int(math.Round(float64(f) * (unitsPerTurn / degreesPerTurn)))
	for candidate := estimate - 2; candidate <= estimate+2; candidate++ {
		if candidate < math.MinInt16 || candidate > math.MaxInt16 {
			continue
		}
		angle := int16(candidate)
		encoded := unpackAngle(angle)
		if math.Float32bits(encoded) == bits {
			return angle, true
		}
	}
	return 0, false
}
