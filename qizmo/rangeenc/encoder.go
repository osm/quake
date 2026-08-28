package rangeenc

import (
	"fmt"
	"math"

	"github.com/osm/quake/qizmo/freq"
)

const renormalizeThreshold = uint32(math.MaxUint32 >> 8)

type Encoder struct {
	low  uint32
	high uint32

	previousLow uint32
	cache       int
	pending     int
	out         []byte
	finished    bool
}

func New() *Encoder {
	return &Encoder{
		high:  math.MaxUint32,
		cache: -1,
	}
}

func (e *Encoder) EncodeSymbol(cumulative []uint32, sym uint32) error {
	if int(sym) >= len(cumulative) {
		return fmt.Errorf("symbol %d outside cumulative table of length %d", sym, len(cumulative))
	}

	var lower uint32
	if sym != 0 {
		lower = cumulative[sym-1]
	}
	return e.encodeInterval(sym, lower, cumulative[sym])
}

func (e *Encoder) EncodeSymbol2x256(
	cumulative0, cumulative1 *[freq.Symbols]uint32,
	sym uint32,
) error {
	// The rows are consecutive halves of one cumulative distribution.
	if sym >= freq.PairedSymbols {
		return fmt.Errorf("symbol %d outside paired cumulative table", sym)
	}

	var lower uint32
	if sym != 0 {
		lower = pairedCumulative(cumulative0, cumulative1, sym-1)
	}
	return e.encodeInterval(sym, lower, pairedCumulative(cumulative0, cumulative1, sym))
}

func (e *Encoder) EncodeFreqByte(ft *freq.Tables, freqTableAddr uint32, value byte) error {
	return e.EncodeSymbol(ft.CumulativeRow(freqTableAddr), uint32(value))
}

func (e *Encoder) EncodeFreqSymbol(
	ft *freq.Tables,
	freqTableAddr uint32,
	symbol uint32,
	symbols uint32,
) error {
	if symbol >= symbols {
		return fmt.Errorf("symbol %d outside frequency model of size %d", symbol, symbols)
	}
	if symbols == freq.PairedSymbols {
		rowIndex := freq.RowIndex(freqTableAddr)
		if rowIndex+1 >= len(ft.Cumulative) {
			return fmt.Errorf("freq table addr %#x has no paired row", freqTableAddr)
		}
		return e.EncodeSymbol2x256(&ft.Cumulative[rowIndex], &ft.Cumulative[rowIndex+1], symbol)
	}
	return e.EncodeSymbol(ft.CumulativeRow(freqTableAddr), symbol)
}

func pairedCumulative(cumulative0, cumulative1 *[freq.Symbols]uint32, index uint32) uint32 {
	if index < freq.Symbols {
		return cumulative0[index]
	}
	return cumulative1[index-freq.Symbols]
}

func (e *Encoder) encodeInterval(sym, lower, upper uint32) error {
	if e.finished {
		return fmt.Errorf("range encoder already finished")
	}
	if upper <= lower {
		return fmt.Errorf("symbol %d has an empty frequency interval", sym)
	}

	rng := e.high - e.low
	low := e.low
	upperOffset := uint32((uint64(upper) * uint64(rng)) >> 32)
	lowerOffset := uint32((uint64(lower) * uint64(rng)) >> 32)
	if upperOffset <= lowerOffset {
		return fmt.Errorf("symbol %d interval is empty at the current range", sym)
	}
	e.high = low + upperOffset - 1
	if sym != 0 {
		e.low = low + lowerOffset
	}
	for e.high-e.low <= renormalizeThreshold {
		e.renormalize()
	}
	return nil
}

func (e *Encoder) Finish() []byte {
	if e.finished {
		return e.out
	}
	e.finished = true

	if e.low <= e.high {
		carry := e.low < e.previousLow
		e.flushCacheAndPending(carry)
		if e.low != 0 {
			e.out = append(e.out, byte(e.high>>24))
		}
	} else {
		// A wrapped final interval carries into the cached byte. Qizmo does
		// not emit the pending bytes in this case.
		e.out = append(e.out, byte(e.cache+1))
	}

	return e.out
}

func (e *Encoder) renormalize() {
	if e.low <= e.high {
		carry := e.low < e.previousLow
		e.flushCacheAndPending(carry)
		e.cache = int(byte(e.low >> 24))
	} else {
		e.pending++
	}

	e.high = (e.high << 8) | math.MaxUint8
	e.low <<= 8
	e.previousLow = e.low
}

func (e *Encoder) flushCacheAndPending(carry bool) {
	if e.cache >= 0 {
		b := byte(e.cache)
		if carry {
			b++
		}
		e.out = append(e.out, b)
	}

	pendingByte := byte(math.MaxUint8)
	if carry {
		pendingByte = 0
	}
	for e.pending != 0 {
		e.out = append(e.out, pendingByte)
		e.pending--
	}
}
