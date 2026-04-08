package quake

import (
	"log"
	"net"

	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/server"
)

type Server struct {
	conn     *net.UDPConn
	logger   *log.Logger
	clients  map[string]*client
	handlers []func(server.Client, packet.Packet) server.HandlerResult
}

func New(logger *log.Logger) Server {
	return Server{
		logger:  logger,
		clients: make(map[string]*client),
	}
}

func (s *Server) HandleFunc(h func(server.Client, packet.Packet) server.HandlerResult) {
	s.handlers = append(s.handlers, h)
}

func (s *Server) Enqueue(cmds []command.Command) {
	for _, c := range s.clients {
		c.cmds = append(c.cmds, cmds...)
	}
}

func (s *Server) EnqueueToClient(addr string, cmds []command.Command) {
	c, ok := s.clients[addr]
	if !ok {
		return
	}
	c.cmds = append(c.cmds, cmds...)
}

func (s *Server) Flush() {
	for _, c := range s.clients {
		if len(c.cmds) == 0 {
			continue
		}
		s.flushClient(c, 0, 0, nil)
	}
}

func (s *Server) FlushClient(addr string) {
	c, ok := s.clients[addr]
	if !ok || len(c.cmds) == 0 {
		return
	}
	s.flushClient(c, 0, 0, nil)
}

func (s *Server) ResetClient(addr string) {
	c, ok := s.clients[addr]
	if !ok {
		return
	}
	c.resetSession(localServerPing)
}

func (s *Server) Close() error {
	if s.conn == nil {
		return nil
	}
	return s.conn.Close()
}

func (s *Server) WriteRawToClients(pkt packet.Packet) {
	if s.conn == nil || pkt == nil {
		return
	}
	buf := pkt.Bytes()
	for _, c := range s.clients {
		if _, err := s.conn.WriteToUDP(buf, c.addr); err != nil {
			s.logger.Printf("unable to write raw packet to client, %v", err)
		}
	}
}

func (s *Server) WriteRawToClient(addr string, pkt packet.Packet) {
	if s.conn == nil || pkt == nil {
		return
	}
	c, ok := s.clients[addr]
	if !ok {
		return
	}
	if _, err := s.conn.WriteToUDP(pkt.Bytes(), c.addr); err != nil {
		s.logger.Printf("unable to write raw packet to client, %v", err)
	}
}
