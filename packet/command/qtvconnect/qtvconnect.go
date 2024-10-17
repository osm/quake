package qtvconnect

import (
	"fmt"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/infostring"
)

type Command struct {
	Version    byte
	Extensions uint32
	Source     string
	UserInfo   *infostring.InfoString
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutBytes([]byte("QTV\n"))
	buf.PutBytes([]byte(fmt.Sprintf("VERSION: %d\n", cmd.Version)))
	buf.PutBytes([]byte(fmt.Sprintf("QTV_EZQUAKE_EXT: %d\n", cmd.Extensions)))
	buf.PutBytes([]byte(fmt.Sprintf("SOURCE: %s\n", cmd.Source)))

	if cmd.UserInfo != nil {
		buf.PutBytes([]byte(fmt.Sprintf("USERINFO: %s\n", string(cmd.UserInfo.Bytes()))))
	}

	buf.PutBytes([]byte("\n"))

	return buf.Bytes()
}
