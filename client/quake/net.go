package quake

import (
	"errors"
	"net"
	"syscall"
	"time"

	"github.com/osm/quake/common/sequencer"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/getchallenge"
	"github.com/osm/quake/packet/command/ip"
	"github.com/osm/quake/packet/command/move"
	"github.com/osm/quake/packet/svc"
)

func (c *Client) Connect(addrPort string) error {
	addr, err := net.ResolveUDPAddr("udp", addrPort)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	c.conn = conn

	c.sendChallenge()

	buf := make([]byte, 1024*64)
	for {

		var isReadTimeout bool
		var incomingSeq uint32
		var incomingAck uint32
		var packet packet.Packet
		var cmds []command.Command

		if err := c.conn.SetReadDeadline(
			time.Now().Add(c.readDeadline),
		); err != nil {
			c.logger.Printf("unable to set read deadline, %v\n", err)
		}

		n, _, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			if errors.Is(err, syscall.ECONNREFUSED) {
				c.logger.Printf("lost connection - reconnecting in 5 seqonds")
				time.Sleep(time.Second * 5)
				c.sendChallenge()
				continue
			}

			if err, ok := err.(net.Error); ok && err.Timeout() {
				if c.seq.GetState() == sequencer.Handshake {
					c.sendChallenge()
				}
				isReadTimeout = true
				goto process
			}

			c.logger.Printf("unexpected read error, %v", err)
			continue
		}

		packet, err = svc.Parse(c.ctx, buf[:n])
		if err != nil {
			c.logger.Printf("error when parsing server data, %v", err)
			continue
		}

		switch p := packet.(type) {
		case *svc.Connectionless:
			cmds = []command.Command{p.Command}
		case *svc.GameData:
			cmds = p.Commands
			incomingSeq = p.Seq
			incomingAck = p.Ack

		}

		for _, h := range c.handlers {
			cmds = append(cmds, h(packet)...)
		}
	process:
		c.processCommands(incomingSeq, incomingAck, cmds, isReadTimeout)
	}
}

func (c *Client) processCommands(
	incomingSeq, incomingAck uint32,
	serverCmds []command.Command,
	isReadTmeout bool,
) {
	var cmds []command.Command

	for _, serverCmd := range serverCmds {
		for _, cmd := range c.handleServerCommand(serverCmd) {
			switch cmd := cmd.(type) {
			case *connect.Command, *ip.Command:
				if _, err := c.conn.Write(
					(&clc.Connectionless{Command: cmd}).Bytes(),
				); err != nil {
					c.logger.Printf("unable to send connectionless command, %v\n", err)
				}
			default:
				cmds = append(cmds, cmd)
			}
		}
	}

	if c.seq.GetState() <= sequencer.Handshake {
		return
	}

	c.cmdsMu.Lock()
	outSeq, outAck, outCmds, err := c.seq.Process(incomingSeq, incomingAck, append(c.cmds, cmds...))
	c.cmds = []command.Command{}
	c.cmdsMu.Unlock()

	if err == sequencer.ErrRateLimit {
		return
	}

	sendCmds := outCmds
	if c.seq.GetState() == sequencer.Connected {
		sendCmds = appendMoveIfMissing(sendCmds, c.getMove(outSeq))
	}

	if _, err := c.conn.Write((&clc.GameData{
		Seq:      outSeq,
		Ack:      outAck,
		QPort:    c.qPort,
		Commands: rewriteMoveChecksums(sendCmds, outSeq),
	}).Bytes()); err != nil {
		c.logger.Printf("unable to send game data, %v\n", err)
	}
}

func (c *Client) sendChallenge() {
	c.seq.Reset()
	c.seq.SetState(sequencer.Handshake)
	c.readDeadline = connectReadDeadline
	if _, err := c.conn.Write(
		(&clc.Connectionless{Command: &getchallenge.Command{}}).Bytes(),
	); err != nil {
		c.logger.Printf("unable to send challenge, %v\n", err)
	}
}

func appendMoveIfMissing(cmds []command.Command, fallback *move.Command) []command.Command {
	for _, cmd := range cmds {
		if _, ok := cmd.(*move.Command); ok {
			return cmds
		}
	}
	return append(cmds, fallback)
}

func rewriteMoveChecksums(cmds []command.Command, outSeq uint32) []command.Command {
	for _, cmd := range cmds {
		mv, ok := cmd.(*move.Command)
		if !ok {
			continue
		}
		mv.Checksum = mv.GetChecksum(outSeq - 1)
	}
	return cmds
}
