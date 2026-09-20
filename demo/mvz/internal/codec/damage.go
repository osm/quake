package codec

import (
	"bytes"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/packet/command/damage"
	"github.com/osm/quake/protocol"
)

type rangeDamage struct {
	armor       byte
	blood       byte
	coordSize   int
	hasPosition bool
	coordinates [3]uint32
}

var damageCoordinateModels = [3][4]frequency.Model{
	{
		frequency.ModelDamageCoordXByte0,
		frequency.ModelDamageCoordXByte1,
		frequency.ModelDamageCoordXByte2,
		frequency.ModelDamageCoordXByte3,
	},
	{
		frequency.ModelDamageCoordYByte0,
		frequency.ModelDamageCoordYByte1,
		frequency.ModelDamageCoordYByte2,
		frequency.ModelDamageCoordYByte3,
	},
	{
		frequency.ModelDamageCoordZByte0,
		frequency.ModelDamageCoordZByte1,
		frequency.ModelDamageCoordZByte2,
		frequency.ModelDamageCoordZByte3,
	},
}

var damageCoordinateLowHighNonzeroModels = [3]frequency.Model{
	frequency.ModelDamageCoordXLoHighNonzero,
	frequency.ModelDamageCoordYLoHighNonzero,
	frequency.ModelDamageCoordZLoHighNonzero,
}

func newDamage(raw []byte, command *damage.Command) *rangeDamage {
	if command == nil || !validCoordSize(int(command.CoordSize)) ||
		!bytes.Equal(raw, command.Bytes()) || len(raw) != 3+3*int(command.CoordSize) {
		return nil
	}
	result := &rangeDamage{armor: raw[1], blood: raw[2], coordSize: int(command.CoordSize)}
	offset := 3
	for axis := range result.coordinates {
		result.coordinates[axis] = readCoord(raw[offset:], result.coordSize)
		if result.coordinates[axis] != 0 {
			result.hasPosition = true
		}
		offset += result.coordSize
	}
	return result
}

func encodeDamage(
	sink symbolSink,
	state *rangeCodecState,
	damage rangeDamage,
) error {
	if err := encodeDamageHeader(sink, damage); err != nil {
		return err
	}
	if !damage.hasPosition {
		return nil
	}
	base, baseKind := selectDamageCoordBase(state, damage)
	if err := sink.Put(frequency.ModelDamageCoordBase, baseKind); err != nil {
		return err
	}
	return encodeDamageCoords(sink, state, damage, base)
}

func decodeDamage(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
) ([]byte, error) {
	damage, err := decodeDamageHeader(decoder, models)
	if err != nil {
		return nil, err
	}
	out := []byte{protocol.SVCDamage, damage.armor, damage.blood}
	if !damage.hasPosition {
		// Damage at position zero does not change the shared coordinate history.
		for range damage.coordinates {
			out = appendCoord(out, 0, damage.coordSize)
		}
		return out, nil
	}
	base, err := decodeDamageCoordBase(decoder, models, state)
	if err != nil {
		return nil, err
	}
	return decodeDamageCoords(decoder, models, state, damage, base, out)
}

func encodeDamageHeader(sink symbolSink, damage rangeDamage) error {
	if err := sink.Put(frequency.ModelDamageArmor, damage.armor); err != nil {
		return err
	}
	if err := sink.Put(frequency.ModelDamageBlood, damage.blood); err != nil {
		return err
	}
	coordSize := byte(damage.coordSize)
	if err := sink.Put(frequency.ModelDamageCoordSize, coordSize); err != nil {
		return err
	}
	hasPosition := byte(0)
	if damage.hasPosition {
		hasPosition = 1
	}
	return sink.Put(frequency.ModelDamageHasPosition, hasPosition)
}

func decodeDamageHeader(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
) (rangeDamage, error) {
	armor, err := decodeSymbol(decoder, models, frequency.ModelDamageArmor)
	if err != nil {
		return rangeDamage{}, err
	}
	blood, err := decodeSymbol(decoder, models, frequency.ModelDamageBlood)
	if err != nil {
		return rangeDamage{}, err
	}
	coordSize, err := decodeSymbol(decoder, models, frequency.ModelDamageCoordSize)
	if err != nil {
		return rangeDamage{}, err
	}
	if !validCoordSize(int(coordSize)) {
		return rangeDamage{}, fmt.Errorf(
			"invalid decoded damage coordinate size %d", coordSize,
		)
	}
	hasPosition, err := decodeSymbol(decoder, models, frequency.ModelDamageHasPosition)
	if err != nil {
		return rangeDamage{}, err
	}
	if hasPosition > 1 {
		return rangeDamage{}, fmt.Errorf("invalid damage position flag %d", hasPosition)
	}
	return rangeDamage{
		armor:       armor,
		blood:       blood,
		coordSize:   int(coordSize),
		hasPosition: hasPosition != 0,
	}, nil
}

func selectDamageCoordBase(
	state *rangeCodecState,
	damage rangeDamage,
) ([3]uint32, byte) {
	base := state.previousEffectCoordinates
	baseKind := effectBasePrevious
	target := state.currentCommandTarget
	if target < rangePlayerSlots {
		candidate := state.players[target].origin
		if preferCoordBase(candidate, base, damage.coordinates, damage.coordSize) {
			base = candidate
			baseKind = effectBaseEntity
		}
	}
	return base, baseKind
}

func decodeDamageCoordBase(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
) ([3]uint32, error) {
	base := state.previousEffectCoordinates
	baseKind, err := decodeSymbol(decoder, models, frequency.ModelDamageCoordBase)
	if err != nil {
		return base, err
	}
	switch baseKind {
	case effectBasePrevious:
		return base, nil
	case effectBaseEntity:
		target := state.currentCommandTarget
		if target >= rangePlayerSlots {
			return base, fmt.Errorf(
				"invalid damage player coordinate reference %d", target,
			)
		}
		return state.players[target].origin, nil
	default:
		return base, fmt.Errorf("invalid damage coordinate base %d", baseKind)
	}
}

func encodeDamageCoords(
	sink symbolSink,
	state *rangeCodecState,
	damage rangeDamage,
	base [3]uint32,
) error {
	for axis, coordinate := range damage.coordinates {
		previous := base[axis]
		symbol := coordSymbol(coordinate, previous, damage.coordSize)
		if err := putCoordBytes(
			sink,
			damageCoordinateModels[axis],
			damageCoordinateLowHighNonzeroModels[axis],
			symbol,
			damage.coordSize,
		); err != nil {
			return err
		}
		state.previousEffectCoordinates[axis] = coordinate
	}
	return nil
}

func decodeDamageCoords(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	damage rangeDamage,
	base [3]uint32,
	out []byte,
) ([]byte, error) {
	for axis := range state.previousEffectCoordinates {
		symbol, err := decodeCoordBytes(
			decoder,
			models,
			damageCoordinateModels[axis],
			damageCoordinateLowHighNonzeroModels[axis],
			damage.coordSize,
		)
		if err != nil {
			return nil, err
		}
		coordinate := restoreCoord(base[axis], symbol, damage.coordSize)
		state.previousEffectCoordinates[axis] = coordinate
		out = appendCoord(out, coordinate, damage.coordSize)
	}
	return out, nil
}
