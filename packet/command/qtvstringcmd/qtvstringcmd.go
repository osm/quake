package qtvstringcmd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/protocol/qtv"
)

type Command struct {
	String string
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint16(uint16(1+2+len(cmd.String)) + 1)
	buf.PutByte(byte(qtv.CLCStringCmd))
	buf.PutString(cmd.String)

	return buf.Bytes()
}
