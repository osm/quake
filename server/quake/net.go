package quake

import (
	"net"
	"time"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/common/sequencer"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/command/s2cconnection"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
)

func (s *Server) ListenAndServe(addrPort string) error {
	addr, err := net.ResolveUDPAddr("udp", addrPort)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn

	buf := make([]byte, 1024*64)
	ctx := context.New(context.WithProtocolVersion(protocol.VersionQW))
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			s.logger.Printf("unable to read from socket, %v", err)
			continue
		}

		packet, err := clc.Parse(ctx, buf[:n])
		if err != nil {
			s.logger.Printf("unable to parse CLC data, %v\n", err)
			continue
		}

		key := clientAddr.String()

		c, ok := s.clients[key]
		if !ok {
			c = &client{
				addr:      clientAddr,
				seq:       sequencer.New(sequencer.WithOutgoingSeq(1)),
				lastWrite: time.Now(),
			}

			s.clients[key] = c
		}

		var clientCmds []command.Command
		var incomingSeq uint32
		var incomingAck uint32

		switch p := packet.(type) {
		case *clc.Connectionless:
			clientCmds = []command.Command{p.Command}
		case *clc.GameData:
			clientCmds = p.Commands
			incomingSeq = p.Seq
			incomingAck = p.Ack
		}

		for _, h := range s.handlers {
			s.Enqueue(h(c, packet))
		}

		s.processCommands(c, incomingSeq, incomingAck, clientCmds)

		for _, c := range s.clients {
			if time.Since(c.lastWrite).Seconds() > float64(5) {
				delete(s.clients, c.addr.String())
			}
		}
	}
}

func (s *Server) processCommands(
	client *client,
	incomingSeq, incomingAck uint32,
	clientCmds []command.Command,
) {
	var cmds []command.Command

	for _, clientCmd := range clientCmds {
		for _, cmd := range s.handleClientCommand(client, clientCmd) {
			switch cmd := cmd.(type) {
			case *s2cchallenge.Command, *s2cconnection.Command:
				if _, err := s.conn.WriteToUDP(
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

	if client.seq.GetState() != sequencer.Connected && client.name != "" {
		client.seq.SetState(sequencer.Connected)

		cmds = append(cmds, &stufftext.Command{
			String: "skins",
		})
	}

	outSeq, outAck, outCmds, err := client.seq.Process(incomingSeq, incomingAck, cmds)
	if err == sequencer.ErrRateLimit {
		return
	}

	if _, err := s.conn.WriteToUDP(
		(&svc.GameData{
			Seq:      outSeq,
			Ack:      outAck,
			Commands: append(outCmds, client.cmds...),
		}).Bytes(),
		client.addr,
	); err != nil {
		s.logger.Printf("unable to write data to socket, %v", err)
	}

	client.cmds = []command.Command{}
}
