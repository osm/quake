package ui

type Color byte

const (
	ColorDefault Color = iota
	ColorOriginal
	ColorWhite
	ColorBrown
)

func (c Color) normalized(fallback Color) Color {
	if c == ColorDefault {
		return fallback
	}
	return c
}

func (c Color) text(text string) string {
	if c != ColorWhite && c != ColorBrown {
		return text
	}
	b := []byte(text)
	for i, ch := range b {
		glyph := ch & 0x7f
		if glyph <= ' ' || glyph >= 0x7f {
			continue
		}
		b[i] = glyph
		if c == ColorBrown {
			b[i] |= 0x80
		}
	}
	return string(b)
}
