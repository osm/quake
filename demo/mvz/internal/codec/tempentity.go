package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/packet/command/tempentity"
	"github.com/osm/quake/protocol"
)

type rangeTempEntity struct {
	typeByte       byte
	count          byte
	entity         uint16
	coordSize      int
	coordinateSets int
	coordinates    [2][3]uint32
}

var tempEntityNumberModels = modelPair{
	frequency.ModelTempEntityLoDelta,
	frequency.ModelTempEntityHiDelta,
}

var tempEntityCoordinateModels = [3][4]frequency.Model{
	{
		frequency.ModelTempEntityCoordXByte0,
		frequency.ModelTempEntityCoordXByte1,
		frequency.ModelTempEntityCoordXByte2,
		frequency.ModelTempEntityCoordXByte3,
	},
	{
		frequency.ModelTempEntityCoordYByte0,
		frequency.ModelTempEntityCoordYByte1,
		frequency.ModelTempEntityCoordYByte2,
		frequency.ModelTempEntityCoordYByte3,
	},
	{
		frequency.ModelTempEntityCoordZByte0,
		frequency.ModelTempEntityCoordZByte1,
		frequency.ModelTempEntityCoordZByte2,
		frequency.ModelTempEntityCoordZByte3,
	},
}

var tempEntityCoordinateLowHighNonzeroModels = [3]frequency.Model{
	frequency.ModelTempEntityCoordXLoHighNonzero,
	frequency.ModelTempEntityCoordYLoHighNonzero,
	frequency.ModelTempEntityCoordZLoHighNonzero,
}

var tempEntityBeamCoordBaseModels = [3]frequency.Model{
	effectClassZero:   frequency.ModelTempEntityBeamCoordBase,
	effectClassPlayer: frequency.ModelTempEntityBeamCoordBasePlayer,
	effectClassOther:  frequency.ModelTempEntityBeamCoordBaseWorld,
}

func tempEntityIsBeam(typeByte byte) bool {
	return typeByte == protocol.TELightning1 ||
		typeByte == protocol.TELightning2 ||
		typeByte == protocol.TELightning3
}

func tempEntityIsCounted(typeByte byte) bool {
	return typeByte == protocol.TEGunshot || typeByte == protocol.TEBlood
}

func newTempEntity(raw []byte, command *tempentity.Command) *rangeTempEntity {
	if command == nil ||
		!validCoordSize(int(command.CoordSize)) ||
		!bytes.Equal(raw, command.Bytes()) || len(raw) < 2 {
		return nil
	}
	result := &rangeTempEntity{
		typeByte:       raw[1],
		coordSize:      int(command.CoordSize),
		coordinateSets: 1,
	}
	offset := 2
	if tempEntityIsBeam(result.typeByte) {
		if len(raw)-offset < 2 {
			return nil
		}
		result.entity = binary.LittleEndian.Uint16(raw[offset : offset+2])
		offset += 2
		result.coordinateSets = 2
	} else if tempEntityIsCounted(result.typeByte) {
		if len(raw)-offset < 1 {
			return nil
		}
		result.count = raw[offset]
		offset++
	}
	if len(raw)-offset != result.coordinateSets*3*result.coordSize {
		return nil
	}
	for set := 0; set < result.coordinateSets; set++ {
		for axis := range result.coordinates[set] {
			result.coordinates[set][axis] = readCoord(raw[offset:], result.coordSize)
			offset += result.coordSize
		}
	}
	return result
}

func encodeTempEntity(
	sink symbolSink,
	state *rangeCodecState,
	entity rangeTempEntity,
) error {
	if err := encodeTempEntityHeader(sink, state, entity); err != nil {
		return err
	}
	base, err := encodeTempEntityCoordBase(sink, state, entity)
	if err != nil {
		return err
	}
	return encodeTempEntityCoords(sink, state, entity, base)
}

func decodeTempEntity(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
) ([]byte, error) {
	entity, err := decodeTempEntityHeader(decoder, models, state)
	if err != nil {
		return nil, err
	}
	coordinateBase, err := decodeTempEntityCoordBase(
		decoder, models, state, entity.typeByte,
	)
	if err != nil {
		return nil, err
	}
	return decodeTempEntityCoords(
		decoder, models, state, entity, coordinateBase,
	)
}

func encodeTempEntityHeader(
	sink symbolSink,
	state *rangeCodecState,
	entity rangeTempEntity,
) error {
	if err := sink.Put(frequency.ModelTempEntityType, entity.typeByte); err != nil {
		return err
	}
	if tempEntityIsCounted(entity.typeByte) {
		if err := sink.Put(frequency.ModelTempEntityCount, entity.count); err != nil {
			return err
		}
	}
	if tempEntityIsBeam(entity.typeByte) {
		symbol := zigzag16(entity.entity - state.previousEffectEntity)
		if err := putWord(sink, tempEntityNumberModels, symbol); err != nil {
			return err
		}
		state.previousEffectEntity = entity.entity
	}
	coordSize := byte(entity.coordSize)
	return sink.Put(frequency.ModelTempEntityCoordSize, coordSize)
}

func decodeTempEntityHeader(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
) (rangeTempEntity, error) {
	typeByte, err := decodeSymbol(decoder, models, frequency.ModelTempEntityType)
	if err != nil {
		return rangeTempEntity{}, err
	}
	entity := rangeTempEntity{typeByte: typeByte, coordinateSets: 1}
	if tempEntityIsCounted(typeByte) {
		entity.count, err = decodeSymbol(decoder, models, frequency.ModelTempEntityCount)
		if err != nil {
			return rangeTempEntity{}, err
		}
	}
	if tempEntityIsBeam(typeByte) {
		symbol, err := decodeWord(decoder, models, tempEntityNumberModels)
		if err != nil {
			return rangeTempEntity{}, err
		}
		state.previousEffectEntity += unzigzag16(symbol)
		entity.entity = state.previousEffectEntity
		entity.coordinateSets = 2
	}
	coordSize, err := decodeSymbol(decoder, models, frequency.ModelTempEntityCoordSize)
	if err != nil {
		return rangeTempEntity{}, err
	}
	if !validCoordSize(int(coordSize)) {
		return rangeTempEntity{}, fmt.Errorf(
			"invalid decoded temp-entity coordinate size %d", coordSize,
		)
	}
	entity.coordSize = int(coordSize)
	return entity, nil
}

func tempEntityPrefix(entity rangeTempEntity) []byte {
	out := []byte{protocol.SVCTempEntity, entity.typeByte}
	if tempEntityIsBeam(entity.typeByte) {
		return binary.LittleEndian.AppendUint16(out, entity.entity)
	}
	if tempEntityIsCounted(entity.typeByte) {
		out = append(out, entity.count)
	}
	return out
}

func encodeTempEntityCoordBase(
	sink symbolSink,
	state *rangeCodecState,
	entity rangeTempEntity,
) ([3]uint32, error) {
	base := state.previousEffectCoordinates
	if !tempEntityIsBeam(entity.typeByte) {
		return base, nil
	}
	baseKind := effectBasePrevious
	if candidate, ok := effectEntityCoords(state, entity.entity); ok &&
		preferCoordBase(candidate, base, entity.coordinates[0], entity.coordSize) {
		base = candidate
		baseKind = effectBaseEntity
	}
	class := effectEntityClass(entity.entity)
	model := tempEntityBeamCoordBaseModels[class]
	if err := sink.Put(model, baseKind); err != nil {
		return base, err
	}
	return base, nil
}

func decodeTempEntityCoordBase(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	typeByte byte,
) ([3]uint32, error) {
	base := state.previousEffectCoordinates
	if !tempEntityIsBeam(typeByte) {
		return base, nil
	}
	class := effectEntityClass(state.previousEffectEntity)
	model := tempEntityBeamCoordBaseModels[class]
	baseKind, err := decodeSymbol(decoder, models, model)
	if err != nil {
		return base, err
	}
	if baseKind == effectBasePrevious {
		return base, nil
	}
	if baseKind != effectBaseEntity {
		return base, fmt.Errorf("invalid temp-entity coordinate base %d", baseKind)
	}
	candidate, ok := effectEntityCoords(state, state.previousEffectEntity)
	if !ok {
		return base, fmt.Errorf(
			"invalid temp-entity coordinate reference %d",
			state.previousEffectEntity,
		)
	}
	return candidate, nil
}

func encodeTempEntityCoords(
	sink symbolSink,
	state *rangeCodecState,
	entity rangeTempEntity,
	base [3]uint32,
) error {
	for set := 0; set < entity.coordinateSets; set++ {
		for axis, coordinate := range entity.coordinates[set] {
			previous := state.previousEffectCoordinates[axis]
			if set == 0 {
				previous = base[axis]
			}
			symbol := coordSymbol(coordinate, previous, entity.coordSize)
			if err := putCoordBytes(
				sink,
				tempEntityCoordinateModels[axis],
				tempEntityCoordinateLowHighNonzeroModels[axis],
				symbol,
				entity.coordSize,
			); err != nil {
				return err
			}
			state.previousEffectCoordinates[axis] = coordinate
		}
	}
	return nil
}

func decodeTempEntityCoords(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	entity rangeTempEntity,
	base [3]uint32,
) ([]byte, error) {
	out := tempEntityPrefix(entity)
	for set := 0; set < entity.coordinateSets; set++ {
		for axis := range state.previousEffectCoordinates {
			coordinate, err := decodeTempEntityCoord(
				decoder,
				models,
				state,
				base,
				set,
				axis,
				entity.coordSize,
			)
			if err != nil {
				return nil, err
			}
			out = appendCoord(out, coordinate, entity.coordSize)
		}
	}
	return out, nil
}

func decodeTempEntityCoord(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	base [3]uint32,
	set, axis, coordSize int,
) (uint32, error) {
	symbol, err := decodeCoordBytes(
		decoder,
		models,
		tempEntityCoordinateModels[axis],
		tempEntityCoordinateLowHighNonzeroModels[axis],
		coordSize,
	)
	if err != nil {
		return 0, err
	}
	previous := state.previousEffectCoordinates[axis]
	if set == 0 {
		previous = base[axis]
	}
	coordinate := restoreCoord(previous, symbol, coordSize)
	state.previousEffectCoordinates[axis] = coordinate
	return coordinate, nil
}
