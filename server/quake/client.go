package quake

import (
	"net"
	"time"

	"github.com/osm/quake/common/sequencer"
	"github.com/osm/quake/packet/command"
)

type client struct {
	addr      *net.UDPAddr
	cmds      []command.Command
	seq       *sequencer.Sequencer
	lastWrite time.Time
	name      string
}

func (c *client) GetName() string {
	return c.name
}
