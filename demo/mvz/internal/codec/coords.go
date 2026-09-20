package codec

import (
	"encoding/binary"
	"math/bits"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
)

func validCoordSize(size int) bool {
	return size == 2 || size == 4
}

func readCoord(data []byte, size int) uint32 {
	if size == 2 {
		return uint32(binary.LittleEndian.Uint16(data))
	}
	return binary.LittleEndian.Uint32(data)
}

func appendCoord(dst []byte, value uint32, size int) []byte {
	if size == 2 {
		return binary.LittleEndian.AppendUint16(dst, uint16(value))
	}
	return binary.LittleEndian.AppendUint32(dst, value)
}

func coordSymbol(value, previous uint32, size int) uint32 {
	if size == 2 {
		return uint32(zigzag16(uint16(value) - uint16(previous)))
	}
	return zigzag32(value - previous)
}

func restoreCoord(previous, symbol uint32, size int) uint32 {
	if size == 2 {
		return uint32(uint16(previous) + unzigzag16(uint16(symbol)))
	}
	return previous + unzigzag32(symbol)
}

// Compare zigzag residual bit lengths as an estimate of compression cost.
// This does not measure encoded size. Ties retain the current base.
// The selected base kind is coded explicitly; decoding never repeats this choice.
func preferCoordBase(
	candidate, current, target [3]uint32,
	coordSize int,
) bool {
	return coordResidualScore(candidate, target, coordSize) <
		coordResidualScore(current, target, coordSize)
}

func coordResidualScore(base, target [3]uint32, coordSize int) int {
	score := 0
	for axis := range target {
		symbol := coordSymbol(target[axis], base[axis], coordSize)
		score += bits.Len32(symbol)
	}
	return score
}

// Write high bytes first so the decoder can select the model for the low byte.
// That model depends only on bits 8 through 15, even for symbols with four bytes in v1.
func putCoordBytes(
	sink symbolSink,
	models [4]frequency.Model,
	lowHighNonzeroModel frequency.Model,
	value uint32,
	size int,
) error {
	for byteIndex := size - 1; byteIndex >= 0; byteIndex-- {
		model := models[byteIndex]
		if byteIndex == 0 && byte(value>>8) != 0 {
			model = lowHighNonzeroModel
		}
		if err := sink.Put(model, byte(value>>uint(byteIndex*8))); err != nil {
			return err
		}
	}
	return nil
}

func decodeCoordBytes(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	byteModels [4]frequency.Model,
	lowHighNonzeroModel frequency.Model,
	size int,
) (uint32, error) {
	var value uint32
	for byteIndex := size - 1; byteIndex >= 0; byteIndex-- {
		model := byteModels[byteIndex]
		if byteIndex == 0 && byte(value>>8) != 0 {
			model = lowHighNonzeroModel
		}
		symbol, err := decodeSymbol(decoder, models, model)
		if err != nil {
			return 0, err
		}
		value |= uint32(symbol) << uint(byteIndex*8)
	}
	return value, nil
}
