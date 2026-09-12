package quake

import (
	"net"
	"sync"
	"time"

	"github.com/osm/quake/common/sequencer"
	"github.com/osm/quake/packet/command"
)

type client struct {
	mu        sync.Mutex
	sendMu    sync.Mutex
	addr      *net.UDPAddr
	cmds      []command.Command
	seq       *sequencer.Sequencer
	lastWrite time.Time
	name      string
}

func (c *client) GetName() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.name
}

func (c *client) GetAddr() string {
	if c.addr == nil {
		return ""
	}
	return c.addr.String()
}

func (c *client) resetSession(ping int16) {
	c.cmds = nil
	c.seq = sequencer.New(sequencer.WithOutgoingSeq(1), sequencer.WithPing(ping))
	c.lastWrite = time.Now()
}
