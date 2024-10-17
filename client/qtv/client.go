package qtv

import (
	"errors"
	"log"
	"net"
	"os"
	"sync"

	"github.com/osm/quake/client"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command"
)

var (
	ErrNoName      = errors.New("no name supplied")
	ErrNoTeam      = errors.New("no team supplied")
	ErrUnknownAddr = errors.New("unknown address format")
)

type Client struct {
	conn     net.Conn
	logger   *log.Logger
	ctx      *context.Context
	handlers []func(packet.Packet) []command.Command
	cmdsMu   sync.Mutex
	cmds     []command.Command

	isHandshaking bool
	isConnected   bool
	serverCount   int32

	name string
	team string
}

func New(name, team string, opts ...Option) (client.Client, error) {
	if name == "" {
		return nil, ErrNoName
	}

	if team == "" {
		return nil, ErrNoTeam
	}

	c := &Client{
		ctx:    context.New(context.WithIsMVD(true)),
		logger: log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime),
		name:   name,
		team:   team,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

func (c *Client) HandleFunc(h func(packet.Packet) []command.Command) {
	c.handlers = append(c.handlers, h)
}

func (c *Client) Enqueue(cmds []command.Command) {
	c.cmdsMu.Lock()
	c.cmds = append(c.cmds, cmds...)
	c.cmdsMu.Unlock()
}

func (c *Client) Quit() {}
