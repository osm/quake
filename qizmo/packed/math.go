package packed

const velocityToCoordDivisor = 125

type WordDelta struct {
	Low  byte
	High byte
}

func SplitWordDelta(target, current int16) WordDelta {
	low, high := SplitDelta16(uint16(target) - uint16(current))
	return WordDelta{Low: low, High: high}
}

func AddWrap16(a, b int16) int16 {
	return int16(uint16(a) + uint16(b))
}

func SplitDelta16(delta uint16) (byte, byte) {
	low := byte(delta)
	high := byte((delta - uint16(int16(int8(low)))) >> 8)
	return low, high
}

func AddLow16(x uint32, d int16) uint32 {
	lo := int16(uint16(x & 0xffff))
	lo += d
	return (x & 0xffff0000) | uint32(uint16(lo))
}

func AddHigh16(x uint32, d int16) uint32 {
	hi := int16(uint16((x >> 16) & 0xffff))
	hi += d
	return (x & 0x0000ffff) | (uint32(uint16(hi)) << 16)
}

func Scaled16(val int16, scale int) int16 {
	v := int(val)
	neg := v < 0
	if neg {
		v = -v
	}
	out := (v*scale + velocityToCoordDivisor/2) / velocityToCoordDivisor
	if neg {
		out = -out
	}
	return int16(out)
}
