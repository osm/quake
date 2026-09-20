package rangecoder

import (
	"fmt"
	"math"
)

const renormalizeThreshold = uint32(math.MaxUint32 >> 8)

// Encoder uses arithmetic coding with uint32 cumulative bounds.
type Encoder struct {
	low  uint32
	high uint32

	previousLow uint32
	cache       int
	pending     int
	out         []byte
	finished    bool
}

// NewEncoder creates an empty range stream.
func NewEncoder() *Encoder {
	return &Encoder{
		high:  math.MaxUint32,
		cache: -1,
	}
}

// EncodeSymbol appends one symbol using a cumulative distribution whose final
// entry is math.MaxUint32.
func (e *Encoder) EncodeSymbol(cumulative []uint32, symbol uint32) error {
	if uint64(symbol) >= uint64(len(cumulative)) {
		return fmt.Errorf(
			"symbol %d outside cumulative table of length %d",
			symbol,
			len(cumulative),
		)
	}

	var lower uint32
	if symbol != 0 {
		lower = cumulative[symbol-1]
	}
	return e.encodeInterval(symbol, lower, cumulative[symbol])
}

func (e *Encoder) encodeInterval(symbol, lower, upper uint32) error {
	if e.finished {
		return fmt.Errorf("range encoder already finished")
	}
	if upper <= lower {
		return fmt.Errorf("symbol %d has an empty frequency interval", symbol)
	}

	rangeWidth := e.high - e.low
	low := e.low
	upperOffset := uint32((uint64(upper) * uint64(rangeWidth)) >> 32)
	lowerOffset := uint32((uint64(lower) * uint64(rangeWidth)) >> 32)
	if upperOffset <= lowerOffset {
		return fmt.Errorf("symbol %d interval is empty at the current range", symbol)
	}
	e.high = low + upperOffset - 1
	if symbol != 0 {
		e.low = low + lowerOffset
	}
	for e.high-e.low <= renormalizeThreshold {
		e.renormalize()
	}
	return nil
}

// Finish returns the complete encoded stream. Repeated calls return the same
// bytes. No symbols may be encoded afterward.
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
		// A wrapped final interval carries into the cached byte. Pending bytes
		// are deliberately omitted in this case.
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
		value := byte(e.cache)
		if carry {
			value++
		}
		e.out = append(e.out, value)
	}

	pendingValue := byte(math.MaxUint8)
	if carry {
		pendingValue = 0
	}
	for e.pending != 0 {
		e.out = append(e.out, pendingValue)
		e.pending--
	}
}
