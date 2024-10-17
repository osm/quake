package quake

import (
	"errors"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/osm/quake/client"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/common/rand"
	"github.com/osm/quake/common/sequencer"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/protocol"
)

var (
	ErrNoName = errors.New("no name supplied")
	ErrNoTeam = errors.New("no team supplied")
)

const connectReadDeadline = time.Duration(13)

type Client struct {
	conn     *net.UDPConn
	logger   *log.Logger
	ctx      *context.Context
	cmdsMu   sync.Mutex
	cmds     []command.Command
	seq      *sequencer.Sequencer
	handlers []func(packet.Packet) []command.Command

	addrPort string
	qPort    uint16

	serverCount  int32
	readDeadline time.Duration

	clientVersion string
	isSpectator   bool
	mapName       string
	name          string
	ping          int16
	team          string
	bottomColor   byte
	topColor      byte

	fteEnabled       bool
	fteExtensions    uint32
	fte2Enabled      bool
	fte2Extensions   uint32
	mvdEnabled       bool
	mvdExtensions    uint32
	zQuakeEnabled    bool
	zQuakeExtensions uint32
}

func New(name, team string, opts ...Option) (client.Client, error) {
	if name == "" {
		return nil, ErrNoName
	}

	if team == "" {
		return nil, ErrNoTeam
	}

	c := &Client{
		ctx:          context.New(context.WithProtocolVersion(protocol.VersionQW)),
		logger:       log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime),
		seq:          sequencer.New(),
		readDeadline: connectReadDeadline,
		qPort:        rand.Uint16(),
		name:         name,
		ping:         999,
		team:         team,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.seq.SetPing(c.ping)

	return c, nil
}

func (c *Client) Enqueue(cmds []command.Command) {
	c.cmdsMu.Lock()
	c.cmds = append(c.cmds, cmds...)
	c.cmdsMu.Unlock()
}

func (c *Client) HandleFunc(h func(h packet.Packet) []command.Command) {
	c.handlers = append(c.handlers, h)
}

func (c *Client) Quit() {
	c.cmds = append(c.cmds, &stringcmd.Command{String: "drop"})

	if c.seq.GetState() == sequencer.Connected {
		time.Sleep(time.Duration(c.ping) * time.Millisecond)
	}
}
