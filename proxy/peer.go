package proxy

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	qizmonet "github.com/osm/quake/net/qizmo"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
)

const defaultQWPort = "27500"

type peerConn interface {
	ReadPacket([]byte) (int, error)
	WritePacket([]byte) error
	SetReadDeadline(time.Time) error
	Close() error
}

type udpPeerConn struct {
	*net.UDPConn
}

func (c *udpPeerConn) ReadPacket(buf []byte) (int, error) {
	n, _, err := c.ReadFromUDP(buf)
	return n, err
}

func (c *udpPeerConn) WritePacket(packet []byte) error {
	_, err := c.Write(packet)
	return err
}

type qizmoPeerConn struct {
	conn          *net.UDPConn
	serverDecoder *qizmonet.ServerDecoder
	clientEncoder *qizmonet.ClientEncoder
	codecMu       sync.Mutex
	writeMu       sync.Mutex
}

func dialPeer(target string, qizmoCompression bool) (peerConn, error) {
	address := peerAddress(target)
	if qizmoCompression {
		return dialQizmoPeer(address)
	}

	udpAddress, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("resolve UDP target: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		return nil, err
	}
	return &udpPeerConn{UDPConn: conn}, nil
}

func peerAddress(target string) string {
	if _, _, err := net.SplitHostPort(target); err == nil {
		return target
	}
	return net.JoinHostPort(strings.Trim(target, "[]"), defaultQWPort)
}

func dialQizmoPeer(address string) (*qizmoPeerConn, error) {
	udpAddress, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("resolve Qizmo UDP target: %w", err)
	}
	conn, err := net.DialUDP("udp", nil, udpAddress)
	if err != nil {
		return nil, err
	}
	return newQizmoPeerConn(conn)
}

func newQizmoPeerConn(conn *net.UDPConn) (*qizmoPeerConn, error) {
	endpoint := qizmonet.NewEndpointState()
	decoder, err := qizmonet.NewServerDecoder(endpoint)
	if err != nil {
		conn.Close()
		return nil, err
	}
	encoder, err := qizmonet.NewClientEncoder(endpoint)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &qizmoPeerConn{
		conn:          conn,
		serverDecoder: decoder,
		clientEncoder: encoder,
	}, nil
}

func (c *qizmoPeerConn) ReadPacket(buf []byte) (int, error) {
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			return 0, err
		}
		compressed := qizmoprotocol.IsCompressedLinkPacket(buf[:n])
		c.codecMu.Lock()
		packet, err := c.serverDecoder.Decode(buf[:n])
		if err == nil && compressed {
			c.clientEncoder.Enable()
		}
		c.codecMu.Unlock()
		if err != nil {
			return 0, err
		}
		if packet == nil {
			continue
		}
		if len(packet) > len(buf) {
			return 0, fmt.Errorf("upstream packet is too large: %d", len(packet))
		}
		return copy(buf, packet), nil
	}
}

func (c *qizmoPeerConn) WritePacket(packet []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.codecMu.Lock()
	c.serverDecoder.ObserveClientPacket(packet)
	encoded, err := c.clientEncoder.Encode(packet)
	c.codecMu.Unlock()
	if err != nil {
		return err
	}
	if _, err := c.conn.Write(encoded); err != nil {
		return err
	}
	return nil
}

func (c *qizmoPeerConn) SetReadDeadline(deadline time.Time) error {
	return c.conn.SetReadDeadline(deadline)
}

func (c *qizmoPeerConn) Close() error {
	return c.conn.Close()
}
