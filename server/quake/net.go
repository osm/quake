package quake

import (
	"errors"
	"net"
	"time"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/common/sequencer"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/command/s2cconnection"
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

	go func() {
		for {
			for _, c := range s.snapshot() {
				c.mu.Lock()
				expired := time.Since(c.lastWrite).Seconds() > float64(5)
				c.mu.Unlock()
				if expired {
					s.mu.Lock()
					delete(s.clients, c.addr.String())
					s.mu.Unlock()
				}
			}

			time.Sleep(time.Second * 60)
		}
	}()

	buf := make([]byte, 1024*64)
	ctx := context.New(context.WithProtocolVersion(protocol.VersionQW))
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			s.logger.Printf("unable to read from socket, %v", err)
			continue
		}

		packet, err := clc.Parse(ctx, buf[:n])
		if err != nil {
			s.logger.Printf("unable to parse CLC data, %v\n", err)
			continue
		}
		key := clientAddr.String()

		s.mu.Lock()
		c, ok := s.clients[key]
		if !ok {
			c = &client{
				addr:      clientAddr,
				seq:       sequencer.New(sequencer.WithOutgoingSeq(1), sequencer.WithPing(localServerPing)),
				lastWrite: time.Now(),
			}

			s.clients[key] = c
		}
		s.mu.Unlock()

		var clientCmds []command.Command
		var incomingSeq uint32
		var incomingAck uint32
		consume := false

		switch p := packet.(type) {
		case *clc.Connectionless:
			clientCmds = []command.Command{p.Command}
		case *clc.GameData:
			clientCmds = p.Commands
			incomingSeq = p.Seq
			incomingAck = p.Ack
		}

		for _, h := range s.handlers {
			res := h(c, packet)
			s.Enqueue(res.Commands)
			consume = consume || res.Consume
		}

		if consume {
			continue
		}

		s.processCommands(c, incomingSeq, incomingAck, clientCmds)
	}
}

func (s *Server) processCommands(
	client *client,
	incomingSeq, incomingAck uint32,
	clientCmds []command.Command,
) {
	client.mu.Lock()
	var cmds []command.Command

	for _, clientCmd := range clientCmds {
		for _, cmd := range s.handleClientCommand(client, clientCmd) {
			switch cmd := cmd.(type) {
			case *s2cchallenge.Command, *s2cconnection.Command:
				if _, err := s.socket().WriteToUDP(
					(&svc.Connectionless{Command: cmd}).Bytes(),
					client.addr,
				); err != nil {
					s.logger.Printf("unable to send command, %v", err)
				}
			default:
				cmds = append(cmds, cmd)
			}
		}
	}

	if client.seq.GetState() != sequencer.Connected && incomingSeq != 0 {
		client.seq.SetState(sequencer.Connected)
	}

	// Connectionless commands send their own replies directly. In raw live mode
	// we must not follow them with a synthetic empty sequenced packet, because
	// that advances the local client's ack state before the upstream server has
	// sent any real sequenced traffic.
	if incomingSeq == 0 && incomingAck == 0 && len(cmds) == 0 && len(client.cmds) == 0 {
		client.mu.Unlock()
		return
	}

	client.mu.Unlock()
	s.flushClient(client, incomingSeq, incomingAck, cmds)
}

func (s *Server) flushClient(
	client *client,
	incomingSeq, incomingAck uint32,
	cmds []command.Command,
) {
	client.sendMu.Lock()
	defer client.sendMu.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()

	outSeq, outAck, outCmds, err := client.seq.Process(incomingSeq, incomingAck, cmds)
	if err == sequencer.ErrRateLimit {
		return
	}

	allCmds := append(outCmds, client.cmds...)

	if _, err := s.socket().WriteToUDP(
		(&svc.GameData{
			Seq:      outSeq,
			Ack:      outAck,
			Commands: allCmds,
		}).Bytes(),
		client.addr,
	); err != nil {
		s.logger.Printf("unable to write data to socket, %v", err)
	}
	client.lastWrite = time.Now()
	client.cmds = []command.Command{}
}
