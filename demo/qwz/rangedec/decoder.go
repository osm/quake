package rangedec

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/osm/quake/demo/qwz/freq"
)

type Decoder struct {
	Low  uint32
	High uint32
	Code uint32

	Buf []byte
	Pos int

	NumberOfCalls int
}

func New(buf []byte) (*Decoder, error) {
	if len(buf) < 4 {
		return nil, fmt.Errorf("buffer too small")
	}
	code := binary.BigEndian.Uint32(buf[:4])
	return &Decoder{Low: 0, High: 0xffffffff, Code: code, Buf: buf, Pos: 4}, nil
}

func (rd *Decoder) renorm() {
	for (rd.High - rd.Low) <= 0x00ffffff {
		rd.High = (rd.High << 8) | 0xff
		rd.Low <<= 8

		var b uint32
		if rd.Pos < len(rd.Buf) {
			b = uint32(rd.Buf[rd.Pos])
			rd.Pos++
		}
		rd.Code = (rd.Code << 8) | b
	}
}

func (rd *Decoder) DecodeSymbol(cum []uint32, step uint32) (uint32, error) {
	rd.renorm()
	rng := rd.High - rd.Low
	if rng == 0 {
		return 0, fmt.Errorf("zero range")
	}

	num := (uint64(rd.Code-rd.Low) << 32) | 0xffffffff
	scaled := uint32(num / uint64(rng))

	var idx uint32
	for step >>= 1; step != 0; step >>= 1 {
		try := idx + step
		if try != 0 && try-1 < uint32(len(cum)) && cum[try-1] <= scaled {
			idx = try
		}
	}

	sym := idx
	if sym >= uint32(len(cum)) {
		sym = uint32(len(cum) - 1)
	}

	var prev uint32
	if sym > 0 {
		prev = cum[sym-1]
	}
	hi := cum[sym]

	rd.High = rd.Low + uint32((uint64(hi)*uint64(rng))>>32) - 1
	rd.Low = rd.Low + uint32((uint64(prev)*uint64(rng))>>32)
	rd.NumberOfCalls++
	return sym, nil
}

// Qizmo mostly uses its own renormalization flow instead of the generic path
// above, and some rows are split across paired 256-entry tables.
func (rd *Decoder) DecodeSymbolQizmo(cum []uint32, step uint32) (uint32, error) {
	for {
		rng := rd.High - rd.Low
		if rng > 0x00ffffff {
			scaled := uint32((((uint64(rd.Code-rd.Low) << 32) | 0xffffffff) / uint64(rng)))
			var idx uint32
			for step >>= 1; step != 0; step >>= 1 {
				try := idx + step
				if try != 0 && int(try-1) < len(cum) && cum[try-1] <= scaled {
					idx = try
				}
			}
			if int(idx) >= len(cum) {
				idx = uint32(len(cum) - 1)
			}
			hi := cum[idx]
			rd.High = rd.Low + uint32((uint64(hi)*uint64(rng))>>32) - 1
			if idx != 0 {
				prev := cum[idx-1]
				rd.Low = rd.Low + uint32((uint64(prev)*uint64(rng))>>32)
			}
			rd.NumberOfCalls++
			return idx, nil
		}
		if rd.Pos >= len(rd.Buf) {
			return 0, io.EOF
		}
		rd.Low <<= 8
		rd.High = (rd.High << 8) | 0xff
		rd.Code = (rd.Code << 8) | uint32(rd.Buf[rd.Pos])
		rd.Pos++
	}
}

func (rd *Decoder) DecodeSymbolQizmo2x256(cum0, cum1 *[256]uint32) (uint32, error) {
	for {
		rng := rd.High - rd.Low
		if rng > 0x00ffffff {
			scaled := uint32((((uint64(rd.Code-rd.Low) << 32) | 0xffffffff) / uint64(rng)))
			var idx uint32
			for step := uint32(0x100); step != 0; step >>= 1 {
				try := idx + step
				if try == 0 || try > 0x200 {
					continue
				}
				var cumTry uint32
				if try <= 0x100 {
					cumTry = cum0[try-1]
				} else {
					cumTry = cum1[try-0x101]
				}
				if cumTry <= scaled {
					idx = try
				}
			}
			if idx >= 0x200 {
				idx = 0x1ff
			}
			var hi uint32
			if idx < 0x100 {
				hi = cum0[idx]
			} else {
				hi = cum1[idx-0x100]
			}
			rd.High = rd.Low + uint32((uint64(hi)*uint64(rng))>>32) - 1
			if idx != 0 {
				var prev uint32
				if idx <= 0x100 {
					prev = cum0[idx-1]
				} else {
					prev = cum1[idx-0x101]
				}
				rd.Low = rd.Low + uint32((uint64(prev)*uint64(rng))>>32)
			}
			rd.NumberOfCalls++
			return idx, nil
		}
		if rd.Pos >= len(rd.Buf) {
			return 0, io.EOF
		}
		rd.Low <<= 8
		rd.High = (rd.High << 8) | 0xff
		rd.Code = (rd.Code << 8) | uint32(rd.Buf[rd.Pos])
		rd.Pos++
	}
}

func (rd *Decoder) DecodeFreqByte(ft *freq.Tables, freqTableAddr uint32) (byte, error) {
	cum := ft.Cumulative[freq.RowIndex(freqTableAddr)][:]
	v, err := rd.DecodeSymbolQizmo(cum, 0x100)
	if err != nil {
		return 0, err
	}
	return byte(v), nil
}

func (rd *Decoder) DecodeFreqSymbol(
	ft *freq.Tables,
	freqTableAddr uint32,
	step uint32,
) (uint32, error) {
	if step == 0x200 {
		r := freq.RowIndex(freqTableAddr)
		if r+1 >= len(ft.Cumulative) {
			return 0, fmt.Errorf(
				"freq table addr %#x has no paired row",
				freqTableAddr,
			)
		}
		return rd.DecodeSymbolQizmo2x256(&ft.Cumulative[r], &ft.Cumulative[r+1])
	}
	cum := ft.Cumulative[freq.RowIndex(freqTableAddr)][:]
	return rd.DecodeSymbolQizmo(cum, step)
}
