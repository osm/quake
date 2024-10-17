package infostring

import (
	"strings"

	"github.com/osm/quake/common/buffer"
)

type InfoString struct {
	Info []Info
}

type Info struct {
	Key   string
	Value string
}

func New(opts ...Option) *InfoString {
	var infoString InfoString

	for _, opt := range opts {
		opt(&infoString)
	}

	return &infoString
}

func (is *InfoString) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(byte('"'))

	for i := 0; i < len(is.Info); i++ {
		buf.PutBytes([]byte("\\" + is.Info[i].Key))
		buf.PutBytes([]byte("\\" + is.Info[i].Value))
	}

	buf.PutByte(byte('"'))

	return buf.Bytes()
}

func Parse(input string) *InfoString {
	var ret InfoString

	trimmed := strings.Trim(input, "\"")
	parts := strings.Split(trimmed, "\\")

	for i := 1; i < len(parts)-1; i += 2 {
		ret.Info = append(ret.Info, Info{parts[i], parts[i+1]})
	}

	return &ret
}

func (is *InfoString) Get(key string) string {
	for i := 0; i < len(is.Info); i++ {
		if is.Info[i].Key == key {
			return is.Info[i].Value
		}
	}

	return ""
}

func (is *InfoString) Set(key, value string) {
	for i := 0; i < len(is.Info); i++ {
		if is.Info[i].Key == key {
			is.Info[i].Value = value
		}
	}
}
