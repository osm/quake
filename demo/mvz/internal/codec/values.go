package codec

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
)

// A model selects a fixed frequency row; a symbol is a byte coded with that row.
// Context selection happens in the field codecs, before this boundary.
// Training counts these same calls so its grammar cannot drift from encoding.
type symbolSink interface {
	Put(frequency.Model, byte) error
}

type rangeSymbolSink struct {
	encoder *rangecoder.Encoder
	models  *frequency.Models
}

func (s *rangeSymbolSink) Put(model frequency.Model, symbol byte) error {
	row, err := s.models.Row(model)
	if err != nil {
		return err
	}
	return s.encoder.EncodeSymbol(row, uint32(symbol))
}

func decodeSymbol(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	model frequency.Model,
) (byte, error) {
	row, err := models.Row(model)
	if err != nil {
		return 0, err
	}
	symbol, err := decoder.DecodeSymbol(row)
	return byte(symbol), err
}

func putUvarint(sink symbolSink, model frequency.Model, value uint64) error {
	var encoded [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(encoded[:], value)
	for _, b := range encoded[:n] {
		if err := sink.Put(model, b); err != nil {
			return err
		}
	}
	return nil
}

func decodeUvarint(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	model frequency.Model,
) (uint64, error) {
	var encoded [binary.MaxVarintLen64]byte
	for i := range encoded {
		b, err := decodeSymbol(decoder, models, model)
		if err != nil {
			return 0, err
		}
		encoded[i] = b
		if b < 0x80 {
			value, n := binary.Uvarint(encoded[:i+1])
			if n <= 0 {
				return 0, fmt.Errorf("invalid MVZ unsigned varint")
			}
			return value, nil
		}
	}
	return 0, fmt.Errorf("MVZ unsigned varint is too long")
}

// Model pairs are indexed low byte, high byte, regardless of wire order.
type modelPair [2]frequency.Model

func putWord(sink symbolSink, models modelPair, value uint16) error {
	if err := sink.Put(models[0], byte(value)); err != nil {
		return err
	}
	return sink.Put(models[1], byte(value>>8))
}

func decodeWord(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	wordModels modelPair,
) (uint16, error) {
	low, err := decodeSymbol(decoder, models, wordModels[0])
	if err != nil {
		return 0, err
	}
	high, err := decodeSymbol(decoder, models, wordModels[1])
	if err != nil {
		return 0, err
	}
	return uint16(low) | uint16(high)<<8, nil
}

// Zigzag helpers treat unsigned values as signed residual bit patterns.
// Arithmetic wraps at the field width.
func zigzag8(value byte) byte {
	signed := int16(int8(value))
	return byte((signed << 1) ^ (signed >> 7))
}

func unzigzag8(value byte) byte {
	unsigned := int16(value)
	signed := (unsigned >> 1) ^ -(unsigned & 1)
	return byte(int8(signed))
}

func zigzag16(value uint16) uint16 {
	signed := int32(int16(value))
	return uint16((signed << 1) ^ (signed >> 15))
}

func unzigzag16(value uint16) uint16 {
	unsigned := int32(value)
	signed := (unsigned >> 1) ^ -(unsigned & 1)
	return uint16(int16(signed))
}

func zigzag32(value uint32) uint32 {
	signed := int64(int32(value))
	return uint32((signed << 1) ^ (signed >> 31))
}

func unzigzag32(value uint32) uint32 {
	unsigned := int64(value)
	signed := (unsigned >> 1) ^ -(unsigned & 1)
	return uint32(int32(signed))
}

func residualMagnitudeContext16(value uint16) int {
	signed := int32(int16(value))
	if signed < 0 {
		signed = -signed
	}
	switch {
	case signed == 0:
		return 0
	case signed <= 16:
		return 1
	case signed <= 256:
		return 2
	default:
		return 3
	}
}

func residualMagnitudeContext8(value byte) int {
	signed := int16(int8(value))
	if signed < 0 {
		signed = -signed
	}
	switch {
	case signed == 0:
		return 0
	case signed <= 4:
		return 1
	case signed <= 32:
		return 2
	default:
		return 3
	}
}
