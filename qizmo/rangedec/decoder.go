package rangedec

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/osm/quake/qizmo/freq"
)

const (
	initialCodeSize      = 4
	renormalizeThreshold = uint32(math.MaxUint32 >> 8)
)

type Decoder struct {
	low  uint32
	high uint32
	code uint32
	buf  []byte
	pos  int
}

func New(buf []byte) (*Decoder, error) {
	if len(buf) < initialCodeSize {
		return nil, fmt.Errorf("buffer too small")
	}
	return newDecoder(buf), nil
}

func newDecoder(buf []byte) *Decoder {
	code := binary.BigEndian.Uint32(buf[:initialCodeSize])
	return &Decoder{high: math.MaxUint32, code: code, buf: buf, pos: initialCodeSize}
}

func NewPadded(buf []byte) *Decoder {
	// Qizmo arithmetic streams permit four zero bytes of EOF padding.
	padded := make([]byte, len(buf)+4)
	copy(padded, buf)
	return newDecoder(padded)
}

func (rd *Decoder) renormalize(strict bool) error {
	for rd.high-rd.low <= renormalizeThreshold {
		if strict && rd.pos >= len(rd.buf) {
			return io.EOF
		}

		rd.high = (rd.high << 8) | math.MaxUint8
		rd.low <<= 8

		var b uint32
		if rd.pos < len(rd.buf) {
			b = uint32(rd.buf[rd.pos])
			rd.pos++
		}
		rd.code = (rd.code << 8) | b
	}
	return nil
}

func (rd *Decoder) DecodeSymbol(cumulative []uint32, step uint32) (uint32, error) {
	return rd.decodeSymbol(cumulative, nil, uint32(len(cumulative)), step, false)
}

func (rd *Decoder) DecodeSymbolStrict(cumulative []uint32, step uint32) (uint32, error) {
	return rd.decodeSymbol(cumulative, nil, uint32(len(cumulative)), step, true)
}

func (rd *Decoder) DecodeSymbolStrict2x256(
	cumulative0, cumulative1 *[freq.Symbols]uint32,
) (uint32, error) {
	return rd.decodeSymbol(cumulative0[:], cumulative1, freq.PairedSymbols, freq.PairedSymbols, true)
}

func (rd *Decoder) decodeSymbol(
	cumulative0 []uint32,
	cumulative1 *[freq.Symbols]uint32,
	symbols uint32,
	step uint32,
	strict bool,
) (uint32, error) {
	if symbols == 0 {
		return 0, fmt.Errorf("empty cumulative table")
	}
	if err := rd.renormalize(strict); err != nil {
		return 0, err
	}

	rng := rd.high - rd.low
	if rng == 0 {
		return 0, fmt.Errorf("zero range")
	}
	numerator := uint64(rd.code-rd.low)<<32 | math.MaxUint32
	scaled := uint32(numerator / uint64(rng))

	var symbol uint32
	for step >>= 1; step != 0; step >>= 1 {
		candidate := symbol + step
		if candidate != 0 && candidate <= symbols &&
			cumulativeAt(cumulative0, cumulative1, candidate-1) <= scaled {
			symbol = candidate
		}
	}
	if symbol >= symbols {
		symbol = symbols - 1
	}

	upper := cumulativeAt(cumulative0, cumulative1, symbol)
	rd.high = rd.low + uint32((uint64(upper)*uint64(rng))>>32) - 1
	if symbol != 0 {
		lower := cumulativeAt(cumulative0, cumulative1, symbol-1)
		rd.low += uint32((uint64(lower) * uint64(rng)) >> 32)
	}
	return symbol, nil
}

func cumulativeAt(cumulative0 []uint32, cumulative1 *[freq.Symbols]uint32, index uint32) uint32 {
	if index < uint32(len(cumulative0)) {
		return cumulative0[index]
	}
	return cumulative1[index-uint32(len(cumulative0))]
}

func (rd *Decoder) DecodeFreqByte(ft *freq.Tables, freqTableAddr uint32) (byte, error) {
	cum := ft.Cumulative[freq.RowIndex(freqTableAddr)][:]
	v, err := rd.DecodeSymbolStrict(cum, freq.Symbols)
	if err != nil {
		return 0, err
	}
	return byte(v), nil
}

func (rd *Decoder) DecodeFreqSymbol(
	ft *freq.Tables,
	freqTableAddr uint32,
	symbols uint32,
) (uint32, error) {
	if symbols == freq.PairedSymbols {
		r := freq.RowIndex(freqTableAddr)
		if r+1 >= len(ft.Cumulative) {
			return 0, fmt.Errorf(
				"freq table addr %#x has no paired row",
				freqTableAddr,
			)
		}
		return rd.DecodeSymbolStrict2x256(&ft.Cumulative[r], &ft.Cumulative[r+1])
	}
	cum := ft.Cumulative[freq.RowIndex(freqTableAddr)][:]
	return rd.DecodeSymbolStrict(cum, symbols)
}
