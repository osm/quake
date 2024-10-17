package rand

import "math/rand"

func Uint16() uint16 {
	return uint16(rand.Uint32() & 0xffff)
}
