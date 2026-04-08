package proxyshell

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/getchallenge"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
)

const upstreamReadDeadline = 3 * time.Second

type upstreamSession struct {
	log  *log.Logger
	conn *net.UDPConn
	ctx  *context.Context
	mu   sync.Mutex
	cmd  *connect.Command
}

func newUpstreamSession(
	logger *log.Logger,
	cmd *connect.Command,
) (*upstreamSession, error) {
	if logger == nil {
		return nil, errors.New("nil logger")
	}

	return &upstreamSession{
		log: logger,
		ctx: context.New(context.WithProtocolVersion(protocol.VersionQW)),
		cmd: cmd,
	}, nil
}

func (u *upstreamSession) Connect(
	addrPort string,
	handle func(packet.Packet),
) error {
	addr, err := net.ResolveUDPAddr("udp", addrPort)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}

	u.mu.Lock()
	u.conn = conn
	u.mu.Unlock()

	if err := u.sendConnectionless(&getchallenge.Command{}); err != nil {
		_ = conn.Close()
		return err
	}

	go u.readLoop(handle)

	return nil
}

func (u *upstreamSession) readLoop(handle func(packet.Packet)) {
	u.mu.Lock()
	conn := u.conn
	u.mu.Unlock()
	if conn == nil {
		return
	}

	buf := make([]byte, 65536)

	for {
		if err := conn.SetReadDeadline(time.Now().Add(upstreamReadDeadline)); err != nil {
			return
		}
		n, err := conn.Read(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}

			u.log.Printf("remote read error: %v", err)

			return
		}

		pkt, err := svc.Parse(u.ctx, buf[:n])
		if err != nil {
			u.log.Printf("upstream parse error: %v", err)
			continue
		}

		if p, ok := pkt.(*svc.Connectionless); ok {
			if cmd, ok := p.Command.(*s2cchallenge.Command); ok {
				u.mu.Lock()
				u.cmd.ChallengeID = cmd.ChallengeID
				connectCmd := *u.cmd
				u.mu.Unlock()

				if err := u.sendConnectionless(&connectCmd); err != nil {
					u.log.Printf("upstream send connect failed: %v", err)
					return
				}
			}
		}

		if handle != nil {
			handle(pkt)
		}
	}
}

func (u *upstreamSession) sendConnectionless(cmd command.Command) error {
	u.mu.Lock()
	conn := u.conn
	u.mu.Unlock()
	if conn == nil {
		return errors.New("nil conn")
	}

	_, err := conn.Write((&clc.Connectionless{Command: cmd}).Bytes())

	return err
}

func (u *upstreamSession) WritePacket(pkt packet.Packet) error {
	u.mu.Lock()
	conn := u.conn
	u.mu.Unlock()
	if conn == nil {
		return errors.New("nil conn")
	}

	if pkt == nil {
		return nil
	}

	_, err := conn.Write(pkt.Bytes())

	return err
}

func (u *upstreamSession) Close() {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.conn != nil {
		_ = u.conn.Close()
		u.conn = nil
	}
}
