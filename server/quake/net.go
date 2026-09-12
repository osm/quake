package quake

import (
	"errors"
	"net"
	"time"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/getchallenge"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/command/s2cconnection"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
)

const localServerPing = 0

func (s *Server) ListenAndServe(addrPort string) error {
	addr, err := net.ResolveUDPAddr("udp", addrPort)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	return s.Serve(conn)
}

func (s *Server) Serve(conn *net.UDPConn) error {
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	defer conn.Close()
	defer func() {
		for _, c := range s.snapshot() {
			s.removeClient(c)
		}
	}()

	buf := make([]byte, 65536)
	ctx := context.New(context.WithProtocolVersion(protocol.VersionQW))
	nextExpiry := time.Now().Add(time.Second)

	for {
		_ = conn.SetReadDeadline(nextExpiry)
		n, addr, err := conn.ReadFromUDP(buf)
		if !time.Now().Before(nextExpiry) {
			s.expireClients()
			nextExpiry = time.Now().Add(time.Second)
		}
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			return err
		}

		pkt, err := clc.Parse(ctx, buf[:n])
		if err != nil {
			s.logger.Printf("unable to parse CLC data: %v", err)
			continue
		}

		s.handlePacket(conn, addr, pkt)
	}
}

func (s *Server) expireClients() {
	for _, c := range s.snapshot() {
		c.mu.Lock()
		expired := time.Since(c.lastRead) > 60*time.Second
		c.mu.Unlock()
		if expired {
			s.removeClient(c)
		}
	}
}

func (s *Server) packetClient(conn *net.UDPConn, addr *net.UDPAddr, pkt packet.Packet) *client {
	c := s.lookup(addr.String())
	oob, ok := pkt.(*clc.Connectionless)
	if !ok {
		return c
	}

	switch oob.Command.(type) {
	case *getchallenge.Command:
		temporary := &client{addr: addr}
		for _, cmd := range s.handleClientCommand(temporary, oob.Command) {
			_, _ = conn.WriteToUDP((&svc.Connectionless{Command: cmd}).Bytes(), addr)
		}
		return nil
	case *connect.Command:
		if c != nil {
			s.removeClient(c)
		}
		c = &client{
			done:     make(chan struct{}),
			addr:     addr,
			lastRead: time.Now(),
		}
		c.resetSession(localServerPing)
		s.mu.Lock()
		s.clients[addr.String()] = c
		s.mu.Unlock()
	}

	return c
}

func (s *Server) handlePacket(conn *net.UDPConn, addr *net.UDPAddr, pkt packet.Packet) {
	c := s.packetClient(conn, addr, pkt)
	if c == nil || !c.acceptPacket(pkt) {
		return
	}

	if game, ok := pkt.(*clc.GameData); ok {
		for _, h := range s.inputHandlers {
			h(c, game)
		}
	}

	consume := false
	var reliable []command.Command
	for _, h := range s.handlers {
		result := h(c, pkt)
		s.Enqueue(result.Commands)
		reliable = append(reliable, result.ClientCommands...)
		consume = consume || result.Consume
	}
	if consume {
		return
	}

	var inputs []command.Command
	switch p := pkt.(type) {
	case *clc.Connectionless:
		inputs = []command.Command{p.Command}
	case *clc.GameData:
		inputs = p.Commands
	}

	commands, dropping := s.dispatchCommands(conn, c, inputs)
	if dropping {
		s.removeClient(c)
		return
	}
	if _, ok := pkt.(*clc.GameData); ok {
		s.flushClient(c, append(reliable, commands...))
	}
}

func (s *Server) dispatchCommands(conn *net.UDPConn, c *client, inputs []command.Command) ([]command.Command, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var reliable []command.Command
	dropping := false
	for _, input := range inputs {
		if str, ok := input.(*stringcmd.Command); ok && str.String == "drop" {
			dropping = true
		}
		for _, output := range s.handleClientCommand(c, input) {
			switch output.(type) {
			case *s2cchallenge.Command, *s2cconnection.Command:
				_, _ = conn.WriteToUDP((&svc.Connectionless{Command: output}).Bytes(), c.addr)
			default:
				reliable = append(reliable, output)
			}
		}
	}

	return reliable, dropping
}

func (s *Server) flushClient(c *client, reliable []command.Command) {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.mu.Lock()
	seq, ack, commands, err := c.seq.Emit(reliable)
	if err != nil {
		c.mu.Unlock()
		return
	}

	commands = append(commands, c.cmds...)
	c.cmds = nil
	c.mu.Unlock()

	p := &svc.GameData{Seq: seq, Ack: ack, Commands: commands}
	for _, h := range s.outputHandlers {
		h(c, p)
	}
	conn := s.socket()
	if conn == nil {
		return
	}

	if _, err := conn.WriteToUDP(p.Bytes(), c.addr); err != nil {
		s.logger.Printf("unable to write data to socket: %v", err)
	}
}
