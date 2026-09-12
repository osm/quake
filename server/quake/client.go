package quake

import (
	"net"
	"sync"
	"time"

	"github.com/osm/quake/common/sequencer"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/protocol"
)

type client struct {
	done         chan struct{}
	mu           sync.Mutex
	sendMu       sync.Mutex
	addr         *net.UDPAddr
	cmds         []command.Command
	seq          *sequencer.Sequencer
	name         string
	slot         byte
	userID       uint32
	scoreboard   [protocol.QWMaxClients]lobbyPlayer
	lastRead     time.Time
	received     bool
	incoming     uint32
	lobbyStage   int
	lobbyStarted time.Time
	zExtensions  int
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
	c.received = false
	c.lobbyStage = lobbyIdle
	c.scoreboard = [protocol.QWMaxClients]lobbyPlayer{}
	c.lobbyStarted = time.Time{}
	c.seq = sequencer.New(sequencer.WithOutgoingSeq(1), sequencer.WithPing(ping))
}

func (c *client) Done() <-chan struct{} {
	return c.done
}

func (c *client) acceptPacket(pkt packet.Packet) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastRead = time.Now()
	game, ok := pkt.(*clc.GameData)
	if !ok {
		return true
	}

	seq := game.Seq & protocol.QWSequenceMask
	distance := (seq - c.incoming) & protocol.QWSequenceMask
	if c.received && (distance == 0 || distance >= (protocol.QWSequenceMask+1)/2) {
		return false
	}

	c.received, c.incoming = true, seq
	c.seq.Acknowledge(game.Seq, game.Ack)
	c.seq.SetState(sequencer.Connected)
	return true
}
