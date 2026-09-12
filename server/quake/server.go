package quake

import (
	"log"
	"net"
	"sync"

	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/server"
)

type Server struct {
	mu                 sync.Mutex
	conn               *net.UDPConn
	logger             *log.Logger
	clients            map[string]*client
	handlers           []func(server.Client, packet.Packet) server.HandlerResult
	inputHandlers      []func(server.Client, *clc.GameData)
	outputHandlers     []func(server.Client, *svc.GameData)
	disconnectHandlers []func(server.Client)
	lobby              *Lobby
	nextUserID         uint32
}

func New(logger *log.Logger) Server {
	return Server{
		logger:  logger,
		clients: make(map[string]*client),
	}
}

// Register handlers before serving: they are read without locking.
func (s *Server) HandleFunc(h func(server.Client, packet.Packet) server.HandlerResult) {
	s.handlers = append(s.handlers, h)
}

func (s *Server) HandleInputFunc(h func(server.Client, *clc.GameData)) {
	s.inputHandlers = append(s.inputHandlers, h)
}

// Flush can invoke output handlers from other goroutines; protect shared callback state.
func (s *Server) HandleOutputFunc(h func(server.Client, *svc.GameData)) {
	s.outputHandlers = append(s.outputHandlers, h)
}

func (s *Server) HandleDisconnectFunc(h func(server.Client)) {
	s.disconnectHandlers = append(s.disconnectHandlers, h)
}

func (s *Server) snapshot() []*client {
	s.mu.Lock()
	defer s.mu.Unlock()

	clients := make([]*client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}

	return clients
}

func (s *Server) lookup(addr string) *client {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.clients[addr]
}

func (s *Server) Enqueue(cmds []command.Command) {
	for _, c := range s.snapshot() {
		c.mu.Lock()
		c.cmds = append(c.cmds, cmds...)
		c.mu.Unlock()
	}
}

func (s *Server) EnqueueToClient(addr string, cmds []command.Command) {
	c := s.lookup(addr)
	if c == nil {
		return
	}

	c.mu.Lock()
	c.cmds = append(c.cmds, cmds...)
	c.mu.Unlock()
}

func (s *Server) Flush() {
	for _, c := range s.snapshot() {
		s.flushClient(c, nil)
	}
}

func (s *Server) FlushClient(addr string) {
	c := s.lookup(addr)
	if c == nil {
		return
	}

	s.flushClient(c, nil)
}

func (s *Server) ResetClient(addr string) {
	c := s.lookup(addr)
	if c == nil {
		return
	}

	c.mu.Lock()
	c.resetSession(localServerPing)
	c.mu.Unlock()
}

func (s *Server) socket() *net.UDPConn {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.conn
}

func (s *Server) Close() error {
	if conn := s.socket(); conn != nil {
		return conn.Close()
	}
	return nil
}

func (s *Server) WriteRawToClients(pkt packet.Packet) {
	for _, c := range s.snapshot() {
		s.WriteRawToClient(c.GetAddr(), pkt)
	}
}

func (s *Server) WriteRawToClient(addr string, pkt packet.Packet) {
	conn := s.socket()
	if conn == nil || pkt == nil {
		return
	}
	c := s.lookup(addr)
	if c == nil {
		return
	}

	if _, err := conn.WriteToUDP(pkt.Bytes(), c.addr); err != nil {
		s.logger.Printf("unable to write raw packet to client, %v", err)
	}
}

func (s *Server) removeClient(c *client) {
	s.mu.Lock()
	if s.clients[c.GetAddr()] != c {
		s.mu.Unlock()
		return
	}

	delete(s.clients, c.GetAddr())
	close(c.done)
	s.mu.Unlock()

	for _, h := range s.disconnectHandlers {
		h(c)
	}
}
