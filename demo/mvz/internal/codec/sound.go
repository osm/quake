package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/packet/command/sound"
	"github.com/osm/quake/protocol"
)

type rangeSound struct {
	channel     uint16
	volume      byte
	attenuation byte
	number      byte
	coordSize   int
	coordinates [3]uint32
}

// The packed channel word holds the entity number in bits 3 through 11.
func (sound rangeSound) entityNumber() uint16 {
	return sound.channel >> 3 & 0x1ff
}

func newSound(raw []byte, command *sound.Command) *rangeSound {
	if command == nil ||
		!validCoordSize(int(command.CoordSize)) ||
		!bytes.Equal(raw, command.Bytes()) || len(raw) < 5 {
		return nil
	}
	offset := 1
	result := &rangeSound{
		channel:   binary.LittleEndian.Uint16(raw[offset : offset+2]),
		coordSize: int(command.CoordSize),
	}
	offset += 2
	if result.channel&protocol.SoundVolume != 0 {
		result.volume = raw[offset]
		offset++
	}
	if result.channel&protocol.SoundAttenuation != 0 {
		result.attenuation = raw[offset]
		offset++
	}
	result.number = raw[offset]
	offset++
	if len(raw)-offset != 3*result.coordSize {
		return nil
	}
	for axis := range result.coordinates {
		result.coordinates[axis] = readCoord(raw[offset:], result.coordSize)
		offset += result.coordSize
	}
	return result
}

func encodeSound(sink symbolSink, state *rangeCodecState, sound rangeSound) error {
	if err := encodeSoundHeader(sink, state, sound); err != nil {
		return err
	}
	entity := sound.entityNumber()
	base, baseKind := selectSoundCoordBase(state, entity, sound)
	baseModel := soundCoordBaseModels[effectEntityClass(entity)]
	if err := sink.Put(baseModel, baseKind); err != nil {
		return err
	}
	if err := encodeSoundCoords(sink, state, sound, base, baseKind); err != nil {
		return err
	}
	state.previousEffectEntity = entity
	return nil
}

func decodeSound(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
) ([]byte, error) {
	sound, err := decodeSoundHeader(decoder, models, state)
	if err != nil {
		return nil, err
	}
	entity := sound.entityNumber()
	base, baseKind, err := decodeSoundCoordBase(
		decoder, models, state, entity,
	)
	if err != nil {
		return nil, err
	}
	out, err := decodeSoundCoords(
		decoder, models, state, soundPrefix(sound), base, baseKind, sound.coordSize,
	)
	if err != nil {
		return nil, err
	}
	state.previousEffectEntity = entity
	return out, nil
}

func encodeSoundHeader(
	sink symbolSink,
	state *rangeCodecState,
	sound rangeSound,
) error {
	channelSymbol := zigzag16(sound.channel - state.previousSoundChannel)
	if err := putWord(sink, soundChannelModels, channelSymbol); err != nil {
		return err
	}
	state.previousSoundChannel = sound.channel
	if sound.channel&protocol.SoundVolume != 0 {
		if err := sink.Put(frequency.ModelSoundVolume, sound.volume); err != nil {
			return err
		}
	}
	if sound.channel&protocol.SoundAttenuation != 0 {
		if err := sink.Put(frequency.ModelSoundAttenuation, sound.attenuation); err != nil {
			return err
		}
	}
	entity := sound.entityNumber()
	if err := encodeSoundNumber(sink, state, entity, sound.number); err != nil {
		return err
	}
	return sink.Put(frequency.ModelSoundCoordSize, byte(sound.coordSize))
}

func decodeSoundHeader(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
) (rangeSound, error) {
	channelSymbol, err := decodeWord(decoder, models, soundChannelModels)
	if err != nil {
		return rangeSound{}, err
	}
	state.previousSoundChannel += unzigzag16(channelSymbol)
	sound := rangeSound{channel: state.previousSoundChannel}
	if sound.channel&protocol.SoundVolume != 0 {
		sound.volume, err = decodeSymbol(decoder, models, frequency.ModelSoundVolume)
		if err != nil {
			return rangeSound{}, err
		}
	}
	if sound.channel&protocol.SoundAttenuation != 0 {
		sound.attenuation, err = decodeSymbol(
			decoder, models, frequency.ModelSoundAttenuation,
		)
		if err != nil {
			return rangeSound{}, err
		}
	}
	entity := sound.entityNumber()
	sound.number, err = decodeSoundNumber(decoder, models, state, entity)
	if err != nil {
		return rangeSound{}, err
	}
	coordSize, err := decodeSymbol(decoder, models, frequency.ModelSoundCoordSize)
	if err != nil {
		return rangeSound{}, err
	}
	if !validCoordSize(int(coordSize)) {
		return rangeSound{}, fmt.Errorf(
			"invalid decoded sound coordinate size %d", coordSize,
		)
	}
	sound.coordSize = int(coordSize)
	return sound, nil
}

func soundPrefix(sound rangeSound) []byte {
	out := binary.LittleEndian.AppendUint16([]byte{protocol.SVCSound}, sound.channel)
	if sound.channel&protocol.SoundVolume != 0 {
		out = append(out, sound.volume)
	}
	if sound.channel&protocol.SoundAttenuation != 0 {
		out = append(out, sound.attenuation)
	}
	return append(out, sound.number)
}

func encodeSoundNumber(
	sink symbolSink,
	state *rangeCodecState,
	entity uint16,
	number byte,
) error {
	numberModel := soundNumberDeltaModels[effectEntityClass(entity)]
	numberSymbol := zigzag8(number - state.previousSoundNumbers[entity])
	if err := sink.Put(numberModel, numberSymbol); err != nil {
		return err
	}
	state.previousSoundNumbers[entity] = number
	return nil
}

func decodeSoundNumber(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	entity uint16,
) (byte, error) {
	numberModel := soundNumberDeltaModels[effectEntityClass(entity)]
	numberSymbol, err := decodeSymbol(decoder, models, numberModel)
	if err != nil {
		return 0, err
	}
	state.previousSoundNumbers[entity] += unzigzag8(numberSymbol)
	return state.previousSoundNumbers[entity], nil
}

func selectSoundCoordBase(
	state *rangeCodecState,
	entity uint16,
	sound rangeSound,
) ([3]uint32, byte) {
	base := state.previousEffectCoordinates
	baseKind := effectBasePrevious
	if candidate, ok := effectEntityCoords(state, entity); ok &&
		preferCoordBase(candidate, base, sound.coordinates, sound.coordSize) {
		base = candidate
		baseKind = effectBaseEntity
	}
	return base, baseKind
}

func decodeSoundCoordBase(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	entity uint16,
) ([3]uint32, byte, error) {
	baseModel := soundCoordBaseModels[effectEntityClass(entity)]
	baseKind, err := decodeSymbol(decoder, models, baseModel)
	if err != nil {
		return [3]uint32{}, 0, err
	}
	base := state.previousEffectCoordinates
	if baseKind == effectBaseEntity {
		candidate, ok := effectEntityCoords(state, entity)
		if !ok {
			return base, 0, fmt.Errorf(
				"invalid sound entity coordinate reference %d", entity,
			)
		}
		base = candidate
	} else if baseKind != effectBasePrevious {
		return base, 0, fmt.Errorf("invalid sound coordinate base %d", baseKind)
	}
	return base, baseKind, nil
}

func encodeSoundCoords(
	sink symbolSink,
	state *rangeCodecState,
	sound rangeSound,
	base [3]uint32,
	baseKind byte,
) error {
	coordinateModels := soundCoordinateModels
	lowHighNonzeroModels := soundCoordinateLowHighNonzeroModels
	if baseKind == effectBaseEntity {
		coordinateModels = soundEntityCoordinateModels
		lowHighNonzeroModels = soundEntityCoordinateLowHighNonzeroModels
	}
	for axis, coordinate := range sound.coordinates {
		previous := base[axis]
		symbol := coordSymbol(coordinate, previous, sound.coordSize)
		if err := putCoordBytes(
			sink,
			coordinateModels[axis],
			lowHighNonzeroModels[axis],
			symbol,
			sound.coordSize,
		); err != nil {
			return err
		}
		state.previousEffectCoordinates[axis] = coordinate
	}
	return nil
}

func decodeSoundCoords(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	out []byte,
	base [3]uint32,
	baseKind byte,
	coordSize int,
) ([]byte, error) {
	coordinateModels := soundCoordinateModels
	lowHighNonzeroModels := soundCoordinateLowHighNonzeroModels
	if baseKind == effectBaseEntity {
		coordinateModels = soundEntityCoordinateModels
		lowHighNonzeroModels = soundEntityCoordinateLowHighNonzeroModels
	}
	for axis := range state.previousEffectCoordinates {
		symbol, err := decodeCoordBytes(
			decoder,
			models,
			coordinateModels[axis],
			lowHighNonzeroModels[axis],
			coordSize,
		)
		if err != nil {
			return nil, err
		}
		coordinate := restoreCoord(base[axis], symbol, coordSize)
		state.previousEffectCoordinates[axis] = coordinate
		out = appendCoord(out, coordinate, coordSize)
	}
	return out, nil
}
