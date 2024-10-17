package ascii

import (
	"strings"
)

func Parse(input string) string {
	var str strings.Builder

	for i := 0; i < len(input); i++ {
		c := input[i]

		if c > 0 && c < 5 {
			str.WriteByte('#')
		} else if c == 5 {
			str.WriteByte('.')
		} else if c > 5 && c < 10 {
			str.WriteByte('#')
		} else if c == 10 {
			str.WriteByte(10)
		} else if c == 11 {
			str.WriteByte('#')
		} else if c > 11 && c < 14 {
			str.WriteByte(' ')
		} else if c > 14 && c < 16 {
			str.WriteByte('.')
		} else if c == 16 {
			str.WriteByte('[')
		} else if c == 17 {
			str.WriteByte(']')
		} else if c > 17 && c < 28 {
			str.WriteByte(c + 30)
		} else if c >= 28 && c < 32 {
			str.WriteByte('.')
		} else if c == 32 {
			str.WriteByte(' ')
		} else if c > 32 && c < 127 {
			str.WriteByte(c)
		} else if c > 127 && c < 129 {
			str.WriteByte('<')
		} else if c == 129 {
			str.WriteByte('=')
		} else if c == 130 {
			str.WriteByte('>')
		} else if c > 130 && c < 133 {
			str.WriteByte('#')
		} else if c == 133 {
			str.WriteByte('.')
		} else if c > 133 && c < 141 {
			str.WriteByte('#')
		} else if c > 141 && c < 144 {
			str.WriteByte('.')
		} else if c == 144 {
			str.WriteByte('[')
		} else if c == 145 {
			str.WriteByte(']')
		} else if c > 145 && c < 156 {
			str.WriteByte(c - 98)
		} else if c == 156 {
			str.WriteByte('.')
		} else if c == 157 {
			str.WriteByte('<')
		} else if c == 158 {
			str.WriteByte('=')
		} else if c == 159 {
			str.WriteByte('>')
		} else if c == 160 {
			str.WriteByte(' ')
		} else {
			str.WriteByte(c - 128)
		}
	}

	return str.String()
}
