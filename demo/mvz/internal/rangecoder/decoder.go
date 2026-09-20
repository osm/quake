package rangecoder

import (
	"fmt"
	"math"
)

const initialCodeSize = 4

// Decoder uses arithmetic decoding with uint32 cumulative bounds.
type Decoder struct {
	low  uint32
	high uint32
	code uint32
	data []byte
	pos  int
}

// NewDecoder initializes a decoder and supplies virtual zero bytes after the
// encoded prefix. The MVZ grammar and limits on decoded size bound that padding.
func NewDecoder(data []byte) *Decoder {
	decoder := &Decoder{
		high: math.MaxUint32,
		data: data,
	}
	for range initialCodeSize {
		decoder.code = decoder.code<<8 | uint32(decoder.readByteOrZero())
	}
	return decoder
}

// DecodeSymbol returns one symbol from a cumulative distribution.
func (d *Decoder) DecodeSymbol(cumulative []uint32) (uint32, error) {
	if len(cumulative) == 0 {
		return 0, fmt.Errorf("empty cumulative table")
	}
	d.renormalize()

	rangeWidth := d.high - d.low
	if rangeWidth == 0 {
		return 0, fmt.Errorf("zero range")
	}
	numerator := uint64(d.code-d.low)<<32 | math.MaxUint32
	scaled := uint32(numerator / uint64(rangeWidth))

	symbol := findSymbol(cumulative, scaled)

	upper := cumulative[symbol]
	upperOffset := uint32((uint64(upper) * uint64(rangeWidth)) >> 32)
	d.high = d.low + upperOffset - 1
	if symbol != 0 {
		lower := cumulative[symbol-1]
		lowerOffset := uint32((uint64(lower) * uint64(rangeWidth)) >> 32)
		d.low += lowerOffset
	}
	return symbol, nil
}

func findSymbol(cumulative []uint32, scaled uint32) uint32 {
	first := 0
	last := len(cumulative)
	for first < last {
		middle := first + (last-first)/2
		if cumulative[middle] <= scaled {
			first = middle + 1
		} else {
			last = middle
		}
	}
	if first == len(cumulative) {
		first--
	}
	return uint32(first)
}

func (d *Decoder) renormalize() {
	for d.high-d.low <= renormalizeThreshold {
		d.high = (d.high << 8) | math.MaxUint8
		d.low <<= 8

		d.code = d.code<<8 | uint32(d.readByteOrZero())
	}
}

func (d *Decoder) readByteOrZero() byte {
	if d.pos >= len(d.data) {
		return 0
	}
	value := d.data[d.pos]
	d.pos++
	return value
}
