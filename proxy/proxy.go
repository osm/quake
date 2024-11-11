package proxy

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/getchallenge"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/command/s2cconnection"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
)

const maxPacketSize = 8192
const maxInjectionSize = 512

type HandlerType byte

const (
	CLC HandlerType = 0
	SVC HandlerType = 1
)

type Proxy struct {
	conn         *net.UDPConn
	logger       *log.Logger
	readDeadline time.Duration
	clientsMu    sync.Mutex
	clients      map[string]*Client
	clcHandlers  []func(*Client, packet.Packet)
	svcHandlers  []func(*Client, packet.Packet)
}

type Client struct {
	addr    *net.UDPAddr
	conn    *net.UDPConn
	connect *connect.Command
	count   int

	CLCInject CommandQueue
	SVCInject CommandQueue
}

func (c *Client) Address() string { return c.addr.String() }

func New(opts ...Option) *Proxy {
	p := &Proxy{
		logger:       log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime),
		clients:      make(map[string]*Client),
		readDeadline: time.Duration(60 * time.Second),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Proxy) HandleFunc(typ HandlerType, h func(*Client, packet.Packet)) {
	switch typ {
	case CLC:
		p.clcHandlers = append(p.clcHandlers, h)
	case SVC:
		p.svcHandlers = append(p.svcHandlers, h)
	}
}

func (p *Proxy) Serve(addrPort string) error {
	addr, err := net.ResolveUDPAddr("udp", addrPort)

	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	p.conn = conn

	ctx := context.New()
	ctx.SetProtocolVersion(protocol.VersionQW)

	buf := make([]byte, 1024*64)
	for {
		n, clientAddr, err := p.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		packet, err := clc.Parse(ctx, buf[:n])
		if err != nil {
			p.logger.Printf("unable to parse CLC data, %v\n", err)
			continue
		}

		switch packet := packet.(type) {
		case *clc.Connectionless:
			switch cmd := packet.Command.(type) {
			case *getchallenge.Command:
				if _, err := p.conn.WriteToUDP((&svc.Connectionless{
					Command: &s2cchallenge.Command{
						ChallengeID: fmt.Sprintf(
							"c-%d",
							uint16(rand.Uint32()&0xffff),
						),
					},
				}).Bytes(), clientAddr); err != nil {
					p.logger.Printf("unable to write s2cchallenge, %v\n", err)
				}
				continue
			case *connect.Command:
				p.handleClientConnect(cmd, clientAddr)
				continue
			}
		}

		client := p.getClient(clientAddr)
		if client == nil {
			continue
		}

		for _, h := range p.clcHandlers {
			h(client, packet)
		}

		if pkg, ok := packet.(*clc.GameData); ok {
			for {
				c := client.CLCInject.Dequeue()
				if c == nil {
					break
				}

				pkg.Commands = append(pkg.Commands, c)
				if len(pkg.Bytes()) >= maxInjectionSize {
					break
				}
			}
		}

		if _, err := client.conn.Write(packet.Bytes()); err != nil {
			p.logger.Printf("unable to write command, %v\n", err)
		}
	}
}

func (p *Proxy) handleClientConnect(cmd *connect.Command, clientAddr *net.UDPAddr) {
	addrPort := cmd.UserInfo.Get("prx")

	if addrPort == "" {
		p.logger.Printf("no target address found in prx userinfo")
		return
	}

	if !strings.Contains(addrPort, ":") {
		addrPort += ":27500"
	}

	if idx := strings.IndexRune(addrPort, '@'); idx != -1 {
		cmd.UserInfo.Set("prx", addrPort[idx+1:])
		addrPort = addrPort[:idx]
	}

	addr, err := net.ResolveUDPAddr("udp", addrPort)
	if err != nil {
		p.logger.Printf("unable to resolve UDP addr, %v", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		p.logger.Printf("unable to initialize target connection to %s, %v", addrPort, err)
		return
	}

	client := p.addClient(clientAddr, conn, cmd)

	if _, err := p.conn.WriteToUDP(
		(&svc.Connectionless{Command: &s2cconnection.Command{}}).Bytes(), client.addr,
	); err != nil {
		p.logger.Printf("unable to send s2cconnectio command, %v\n", err)
	}

	if _, err := conn.Write(
		(&clc.Connectionless{Command: &getchallenge.Command{}}).Bytes(),
	); err != nil {
		p.logger.Printf("unable to send getchallenge command, %v\n", err)
	}

	go p.handlePeer(client)

	name := cmd.UserInfo.Get("name")
	p.logger.Printf("%s (%s) connected to %s", name, client.addr.String(), addrPort)
}

func (p *Proxy) handlePeer(client *Client) {
	buf := make([]byte, 1024*64)
	ctx := context.New()

	for {
		if err := client.conn.SetReadDeadline(time.Now().Add(p.readDeadline)); err != nil {
			p.logger.Printf("unable to set read deadline, %v\n", err)
			continue
		}
		n, _, err := client.conn.ReadFromUDP(buf)
		if err != nil {
			if client.count > 1 {
				client.count--
				return
			}

			p.removeClient(client)
			return
		}

		packet, err := svc.Parse(ctx, buf[:n])
		if err != nil {
			p.logger.Printf("unable to parse SVC data, %v\n", err)
			continue
		}

		switch packet := packet.(type) {
		case *svc.Connectionless:
			switch cmd := packet.Command.(type) {
			case *s2cchallenge.Command:
				client.connect.ChallengeID = cmd.ChallengeID
				if _, err := client.conn.Write(
					(&svc.Connectionless{Command: client.connect}).Bytes(),
				); err != nil {
					p.logger.Printf("unable to send s2cchallenge command, %v\n", err)
				}
				continue
			case *s2cconnection.Command:
				continue
			}
		}

		for _, h := range p.svcHandlers {
			h(client, packet)
		}

		if pkg, ok := packet.(*svc.GameData); ok {
			for {
				c := client.SVCInject.Dequeue()
				if c == nil {
					break
				}

				pkg.Commands = append(pkg.Commands, c)
				if len(pkg.Bytes()) >= maxInjectionSize {
					break
				}
			}
		}

		if _, err := p.conn.WriteToUDP(packet.Bytes(), client.addr); err != nil {
			p.logger.Printf("unable to write data, %v\n", err)
		}
	}
}

func (p *Proxy) addClient(addr *net.UDPAddr, conn *net.UDPConn, cmd *connect.Command) *Client {
	p.clientsMu.Lock()
	defer p.clientsMu.Unlock()

	key := addr.String()
	client := p.clients[key]
	if client == nil {
		p.clients[key] = &Client{}
	}

	p.clients[key].addr = addr
	p.clients[key].conn = conn
	p.clients[key].connect = cmd
	p.clients[key].count += 1

	return p.clients[key]
}

func (p *Proxy) getClient(addr *net.UDPAddr) *Client {
	p.clientsMu.Lock()
	defer p.clientsMu.Unlock()
	return p.clients[addr.String()]
}

func (p *Proxy) removeClient(client *Client) {
	p.logger.Printf("removing client with address %s", client.addr.String())
	p.clientsMu.Lock()
	defer p.clientsMu.Unlock()
	delete(p.clients, client.addr.String())
}
