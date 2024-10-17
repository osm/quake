package protocol

import (
	"fmt"

	"github.com/osm/quake/common/buffer"
)

type Extension struct {
	Version    uint32
	Extensions uint32
}

func (cmd *Extension) Bytes() []byte {
	buf := buffer.New()

	buf.PutBytes([]byte(fmt.Sprintf("0x%x", cmd.Version)))
	buf.PutBytes([]byte(fmt.Sprintf("0x%x", cmd.Extensions)))
	buf.PutByte(0x0a)

	return buf.Bytes()
}
