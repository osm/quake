package qwd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/protocol"
)

type Data struct {
	Timestamp float32
	Command   byte
	Cmd       *Cmd
	Read      *Read
	Set       *Set
}

func (d *Data) Bytes() []byte {
	buf := buffer.New()

	buf.PutFloat32(d.Timestamp)
	buf.PutByte(d.Command)

	switch d.Command {
	case protocol.DemoCmd:
		buf.PutBytes(d.Cmd.Bytes())
	case protocol.DemoRead:
		buf.PutBytes(d.Read.Bytes())
	case protocol.DemoSet:
		buf.PutBytes(d.Set.Bytes())
	}

	return buf.Bytes()
}
