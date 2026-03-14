package packed

func AddWrap16(a, b int16) int16 {
	return int16(uint16(a) + uint16(b))
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

func SetLow16(x uint32, v int16) uint32 {
	return (x & 0xffff0000) | uint32(uint16(v))
}

func SetHigh16(x uint32, v int16) uint32 {
	return (x & 0x0000ffff) | (uint32(uint16(v)) << 16)
}

func Scaled16(val int16, scale int) int16 {
	v := int(val)
	neg := v < 0
	if neg {
		v = -v
	}
	out := (v*scale + 0x3e) / 0x7d
	if neg {
		out = -out
	}
	return int16(out)
}
