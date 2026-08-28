package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"sync"

	qizmonet "github.com/osm/quake/net/qizmo"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
)

const maxUDPPacketSize = math.MaxUint16

type linkRole string

const (
	clientRole linkRole = "client"
	serverRole linkRole = "server"
)

type linkCodecs interface {
	towardServer([]byte) ([]byte, error)
	towardClient([]byte) ([]byte, error)
}

// Each endpoint locks its two codecs together because the opposite UDP
// directions share Qizmo's client-movement clock.
type clientEndpointCodecs struct {
	mu sync.Mutex

	clientEncoder *qizmonet.ClientEncoder
	serverDecoder *qizmonet.ServerDecoder
}

type serverEndpointCodecs struct {
	mu sync.Mutex

	clientDecoder *qizmonet.ClientDecoder
	serverEncoder *qizmonet.ServerEncoder
	peerHandshake *qizmonet.ServerPeerHandshake
}

func newLinkCodecs(role linkRole, peerHandshake *qizmonet.ServerPeerHandshake) (linkCodecs, error) {
	switch role {
	case clientRole:
		if peerHandshake != nil {
			return nil, fmt.Errorf("qizmo peer mode requires the server role")
		}
		endpoint := qizmonet.NewEndpointState()
		clientEncoder, err := qizmonet.NewClientEncoder(endpoint)
		if err != nil {
			return nil, err
		}
		serverDecoder, err := qizmonet.NewServerDecoder(endpoint)
		if err != nil {
			return nil, err
		}
		return &clientEndpointCodecs{
			clientEncoder: clientEncoder,
			serverDecoder: serverDecoder,
		}, nil

	case serverRole:
		endpoint := qizmonet.NewEndpointState()
		clientDecoder, err := qizmonet.NewClientDecoder(endpoint)
		if err != nil {
			return nil, err
		}
		serverEncoder, err := qizmonet.NewServerEncoder(endpoint)
		if err != nil {
			return nil, err
		}
		return &serverEndpointCodecs{
			clientDecoder: clientDecoder,
			serverEncoder: serverEncoder,
			peerHandshake: peerHandshake,
		}, nil

	default:
		return nil, fmt.Errorf("invalid role %q: use client or server", role)
	}
}

func (c *clientEndpointCodecs) towardServer(packet []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.serverDecoder.ObserveClientPacket(packet)
	return c.clientEncoder.Encode(packet)
}

func (c *clientEndpointCodecs) towardClient(packet []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	compressed := qizmoprotocol.IsCompressedLinkPacket(packet)
	decoded, err := c.serverDecoder.Decode(packet)
	if err == nil && compressed {
		c.clientEncoder.Enable()
	}
	return decoded, err
}

func (c *serverEndpointCodecs) towardServer(packet []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	decoded, _, err := c.clientDecoder.DecodeWithExtensions(packet)
	if err != nil {
		return nil, err
	}
	if c.peerHandshake != nil {
		decoded = c.peerHandshake.ConsumeResponse(decoded)
	}
	c.serverEncoder.ObserveClientPacket(decoded)
	return decoded, nil
}

func (c *serverEndpointCodecs) towardClient(packet []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.peerHandshake != nil {
		advertised, ok := c.peerHandshake.Advertise(packet)
		if ok {
			// Peer identification must be visible before original Qizmo enables
			// compression, so this one packet is intentionally sent raw.
			if err := c.serverEncoder.ObserveRawPacket(advertised); err != nil {
				return nil, err
			}
			return advertised, nil
		}
	}
	return c.serverEncoder.Encode(packet)
}

type forwarder struct {
	downstream *net.UDPConn
	upstream   *net.UDPConn
	codecs     linkCodecs

	clientMu sync.RWMutex
	client   *net.UDPAddr
}

func (f *forwarder) run() error {
	errCh := make(chan error, 2)
	go func() { errCh <- f.forwardTowardClient() }()
	go func() { errCh <- f.forwardTowardServer() }()
	return <-errCh
}

func (f *forwarder) forwardTowardServer() error {
	buffer := make([]byte, maxUDPPacketSize)
	for {
		n, client, err := f.downstream.ReadFromUDP(buffer)
		if err != nil {
			return fmt.Errorf("read downstream: %w", err)
		}
		if !f.acceptClient(client) {
			continue
		}
		packet, err := f.codecs.towardServer(buffer[:n])
		if err != nil {
			return fmt.Errorf("transform downstream packet: %w", err)
		}
		if packet == nil {
			continue
		}
		if _, err := f.upstream.Write(packet); err != nil {
			return fmt.Errorf("write upstream: %w", err)
		}
	}
}

func (f *forwarder) forwardTowardClient() error {
	buffer := make([]byte, maxUDPPacketSize)
	for {
		n, err := f.upstream.Read(buffer)
		if err != nil {
			return fmt.Errorf("read upstream: %w", err)
		}
		packet, err := f.codecs.towardClient(buffer[:n])
		if err != nil {
			return fmt.Errorf("transform upstream packet: %w", err)
		}
		if packet == nil {
			continue
		}
		client := f.currentClient()
		if client == nil {
			continue
		}
		if _, err := f.downstream.WriteToUDP(packet, client); err != nil {
			return fmt.Errorf("write downstream: %w", err)
		}
	}
}

func (f *forwarder) acceptClient(client *net.UDPAddr) bool {
	f.clientMu.Lock()
	defer f.clientMu.Unlock()
	if f.client == nil {
		f.client = client
	}
	return f.client.String() == client.String()
}

func (f *forwarder) currentClient() *net.UDPAddr {
	f.clientMu.RLock()
	defer f.clientMu.RUnlock()
	return f.client
}

func main() {
	listenAddress := flag.String("listen-addr", "127.0.0.1:30000", "downstream UDP address")
	upstreamAddress := flag.String("upstream-addr", "127.0.0.1:30001", "fixed upstream UDP address")
	roleName := flag.String("role", string(clientRole), "link endpoint: client or server")
	qizmoPeer := flag.Bool("qizmo-peer", false, "identify the server endpoint to an original Qizmo peer")
	flag.Parse()

	var peerHandshake *qizmonet.ServerPeerHandshake
	if *qizmoPeer {
		var err error
		peerHandshake, err = qizmonet.NewServerPeerHandshake()
		if err != nil {
			log.Fatal(err)
		}
	}
	codecs, err := newLinkCodecs(linkRole(*roleName), peerHandshake)
	if err != nil {
		log.Fatal(err)
	}
	listenAddr, err := net.ResolveUDPAddr("udp", *listenAddress)
	if err != nil {
		log.Fatalf("resolve listen address: %v", err)
	}
	upstreamAddr, err := net.ResolveUDPAddr("udp", *upstreamAddress)
	if err != nil {
		log.Fatalf("resolve upstream address: %v", err)
	}
	downstream, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	upstream, err := net.DialUDP("udp", nil, upstreamAddr)
	if err != nil {
		log.Fatalf("dial upstream: %v", err)
	}
	defer downstream.Close()
	defer upstream.Close()

	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Printf("%s endpoint: %s -> %s", *roleName, downstream.LocalAddr(), upstream.RemoteAddr())
	if err := (&forwarder{
		downstream: downstream,
		upstream:   upstream,
		codecs:     codecs,
	}).run(); err != nil {
		logger.Fatal(err)
	}
}
