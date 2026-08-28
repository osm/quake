package message

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/rangeenc"
)

const (
	soundChannelSize      = 2
	soundIndexSize        = 1
	soundFixedPayloadSize = soundChannelSize + soundIndexSize + coordinateTripletSize
)

func (e *Encoder) soundOperationLength(data []byte) (int, bool) {
	if len(data) < 1+soundFixedPayloadSize || data[0] != protocol.SVCSound {
		return 0, false
	}
	channel := binary.LittleEndian.Uint16(data[1:3])
	optionalSize := 0
	if channel&protocol.SoundVolume != 0 {
		optionalSize++
	}
	if channel&protocol.SoundAttenuation != 0 {
		optionalSize++
	}
	length := 1 + soundFixedPayloadSize + optionalSize
	if length > len(data) {
		return 0, false
	}
	soundOffset := 1 + soundChannelSize + optionalSize
	if _, ok := e.state.SoundRemapIndex(data[soundOffset]); !ok {
		return 0, false
	}
	return length, true
}

func tempEntityOperationLength(data []byte) (int, bool) {
	if len(data) < 2 || data[0] != protocol.SVCTempEntity {
		return 0, false
	}
	length := 2 + tempEntityPayloadSize(tempEntityShapeForType(data[1]))
	return length, length <= len(data)
}

func (e *Encoder) encodeSVCVoice(enc *rangeenc.Encoder, data []byte) error {
	for i, value := range data {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCVoiceData+uint32(i)*freq.RowSize, value); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeSVCSound(enc *rangeenc.Encoder, ctx *encodingContext, data []byte) error {
	channel := binary.LittleEndian.Uint16(data[0:2])
	entity := channel >> soundEntityShift & soundEntityMask
	channelXOR := channel ^ ctx.lastEntity<<soundEntityShift
	if err := e.encodeRows(enc, []byte{
		byte(channelXOR),
		byte(channelXOR >> 8),
	}, freq.SVCSoundChannelLo, freq.SVCSoundChannelHi); err != nil {
		return err
	}
	off := soundChannelSize
	if channel&protocol.SoundVolume != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCSoundVolume, data[off]); err != nil {
			return err
		}
		off++
	}
	if channel&protocol.SoundAttenuation != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCSoundAttenuation, data[off]); err != nil {
			return err
		}
		off++
	}
	remapIndex, ok := e.state.SoundRemapIndex(data[off])
	if !ok {
		return fmt.Errorf("sound %d has no remap", data[off])
	}
	if err := enc.EncodeFreqByte(e.ft, freq.SVCSoundIndex, remapIndex); err != nil {
		return err
	}
	off++

	runningBase := ctx.lastCoordinates
	target := readCoordinateTriplet(data[off:])
	candidate := newCoordinateCandidate(0, target, runningBase)
	switch {
	case target == runningBase:
		candidate = newCoordinateCandidate(coordinateLastReference, target, runningBase)
	case entity != 0 && entity <= maxPlayers && len(ctx.base.players) != 0:
		coordinates, ok := playerCoordinates(ctx.base.players, entity)
		base := signedCoordinates(coordinates)
		if ok && base == target {
			candidate = newCoordinateCandidate(soundBasePlayerReference, target, base)
		} else if coordinates, ok := playerCoordinates(ctx.currentPlayers, entity); ok {
			candidate = newCoordinateCandidate(0, target, signedCoordinates(coordinates))
		}
	case entity > maxPlayers:
		if coordinates, ok := soundEntityCoordinates(e.state, ctx.currentEntities, entity); ok {
			candidate = newCoordinateCandidate(0, target, signedCoordinates(coordinates))
		}
	}
	if err := e.encodeCoordinateCandidate(enc, freq.SVCSoundFlags, candidate); err != nil {
		return err
	}
	ctx.lastEntity = entity
	ctx.lastCoordinates = target
	return nil
}

func (e *Encoder) encodeSVCMuzzleFlash(
	enc *rangeenc.Encoder,
	ctx *encodingContext,
	data []byte,
) error {
	entity := binary.LittleEndian.Uint16(data)
	xor := entity ^ ctx.lastEntity
	if err := e.encodeRows(enc, []byte{byte(xor), byte(xor >> 8)},
		freq.SVCTEntBeamEntityLo, freq.SVCTEntBeamEntityHi); err != nil {
		return err
	}
	ctx.lastEntity = entity
	return nil
}

func (e *Encoder) encodeSVCDamage(enc *rangeenc.Encoder, ctx *encodingContext, data []byte) error {
	if err := e.encodeRows(enc, data[:2], freq.SVCDamageArmor, freq.SVCDamageBlood); err != nil {
		return err
	}
	hasCoordinates := false
	for _, value := range data[2:] {
		if value != 0 {
			hasCoordinates = true
			break
		}
	}
	if !hasCoordinates {
		return enc.EncodeSymbol(e.ft.CumulativeRow(freq.SVCDamageHasFrom), 0)
	}
	if err := enc.EncodeSymbol(e.ft.CumulativeRow(freq.SVCDamageHasFrom), 1); err != nil {
		return err
	}
	base := ctx.primary.origin
	target := readCoordinateTriplet(data[2:])
	for i, rows := range damageCoordinateDeltaRows {
		delta := packed.SplitWordDelta(target[i], base[i])
		if err := enc.EncodeFreqByte(e.ft, rows.low, delta.Low); err != nil {
			return err
		}
		if err := enc.EncodeFreqByte(e.ft, rows.high, delta.High); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeSVCTempEntity(enc *rangeenc.Encoder, ctx *encodingContext, data []byte) error {
	typeByte := data[0]
	if err := enc.EncodeFreqByte(e.ft, freq.SVCTEntType, typeByte); err != nil {
		return err
	}
	off := 1
	base := ctx.lastCoordinates
	entity := ctx.lastEntity
	switch tempEntityShapeForType(typeByte) {
	case tempEntityCountedPoint:
		if err := enc.EncodeFreqByte(e.ft, freq.SVCTEntCount, data[off]); err != nil {
			return err
		}
		off++
	case tempEntityBeam:
		targetEntity := binary.LittleEndian.Uint16(data[off : off+2])
		off += 2
		start := readCoordinateTriplet(data[off:])
		off += coordinateTripletSize
		entityFlag := byte(0)
		if targetEntity != entity {
			entityFlag = beamEntityDelta
		}
		candidate := newCoordinateCandidate(entityFlag, start, base)
		if start == base {
			candidate = newCoordinateCandidate(coordinateLastReference|entityFlag, start, base)
		} else if targetEntity != 0 {
			if targetEntity <= maxPlayers && len(ctx.base.players) != 0 {
				if coordinates, ok := playerCoordinates(ctx.currentPlayers, targetEntity); ok {
					candidate = newCoordinateCandidate(entityFlag, start, signedCoordinates(coordinates))
				}
			} else if targetEntity > maxPlayers {
				if coordinates, found, _ := entityCoordinates(ctx.base.entities, targetEntity); found {
					candidate = newCoordinateCandidate(entityFlag, start, signedCoordinates(coordinates))
				}
			}
		}
		if err := e.encodeBeamCoordinateCandidate(enc, entity, targetEntity, candidate); err != nil {
			return err
		}
		entity = targetEntity
		base = start
	}

	target := readCoordinateTriplet(data[off:])
	deltas := coordinateDeltas(target, base)
	for i, rows := range tempEntityCoordinateDeltaRows {
		if err := enc.EncodeFreqByte(e.ft, rows.low, deltas[i].Low); err != nil {
			return err
		}
		if err := enc.EncodeFreqByte(e.ft, rows.high, deltas[i].High); err != nil {
			return err
		}
	}
	ctx.lastEntity = entity
	ctx.lastCoordinates = target
	return nil
}

func coordinateDeltas(target, base [3]int16) [3]packed.WordDelta {
	var deltas [3]packed.WordDelta
	for axis := range deltas {
		deltas[axis] = packed.SplitWordDelta(target[axis], base[axis])
	}
	return deltas
}

type coordinateCandidate struct {
	flags  byte
	deltas [3]packed.WordDelta
}

func newCoordinateCandidate(flags byte, target, base [3]int16) coordinateCandidate {
	deltas := coordinateDeltas(target, base)
	setCoordinateMask(&flags, deltas)
	return coordinateCandidate{flags: flags, deltas: deltas}
}

func (e *Encoder) encodeCoordinateCandidate(
	enc *rangeenc.Encoder,
	flagsRow uint32,
	candidate coordinateCandidate,
) error {
	if err := enc.EncodeFreqByte(e.ft, flagsRow, candidate.flags); err != nil {
		return err
	}
	return e.encodeCoordinateDeltas(enc, candidate.deltas)
}

func (e *Encoder) encodeBeamCoordinateCandidate(
	enc *rangeenc.Encoder,
	previousEntity uint16,
	entity uint16,
	candidate coordinateCandidate,
) error {
	if err := enc.EncodeFreqByte(e.ft, freq.SVCTEntBeamFlags, candidate.flags); err != nil {
		return err
	}
	if candidate.flags&beamEntityDelta != 0 {
		xor := entity ^ previousEntity
		if err := e.encodeRows(enc, []byte{byte(xor), byte(xor >> 8)},
			freq.SVCTEntBeamEntityLo, freq.SVCTEntBeamEntityHi); err != nil {
			return err
		}
	}
	return e.encodeCoordinateDeltas(enc, candidate.deltas)
}

func setCoordinateMask(mask *byte, deltas [3]packed.WordDelta) {
	for axis, delta := range deltas {
		setWordDeltaMask(mask, delta, axis)
	}
}

func (e *Encoder) encodeCoordinateDeltas(enc *rangeenc.Encoder, deltas [3]packed.WordDelta) error {
	for i, rows := range coordinateDeltaRows {
		if err := e.encodeWordDelta(enc, deltas[i], rows.low, rows.high); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeSVCNails(enc *rangeenc.Encoder, ctx *encodingContext, data []byte) error {
	count := data[0]
	if err := enc.EncodeFreqByte(e.ft, freq.SVCNailsProjectileCount, count); err != nil {
		return err
	}
	previous := nailProjectileBase(unsignedCoordinates(ctx.primary.origin))
	projectiles := data[1:]
	for projectile := 0; projectile < int(count); projectile++ {
		for field := range previous {
			target := projectiles[projectile*len(nailProjectileDeltaRows)+field]
			delta := target - previous[field]
			if err := enc.EncodeFreqByte(e.ft, nailProjectileDeltaRows[field], delta); err != nil {
				return err
			}
			previous[field] = target
		}
	}
	return nil
}
