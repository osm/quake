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
	handlers []func(server.Client, packet.Packet) []command.Command
}

func New(logger *log.Logger) Server {
	return Server{
		logger:  logger,
		clients: make(map[string]*client),
	}
}

func (s *Server) HandleFunc(h func(server.Client, packet.Packet) []command.Command) {
	s.handlers = append(s.handlers, h)
}

func (s *Server) Enqueue(cmds []command.Command) {
	for _, c := range s.clients {
		c.cmds = append(c.cmds, cmds...)
	}
}
