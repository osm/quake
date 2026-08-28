package qizmo

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
)

type ServerPeerHandshake struct {
	advertisement   string
	responseCommand []byte
	advertised      bool
}

func NewServerPeerHandshake() (*ServerPeerHandshake, error) {
	first, err := newPeerToken()
	if err != nil {
		return nil, fmt.Errorf("create first Qizmo peer token: %w", err)
	}
	second, err := newPeerToken()
	if err != nil {
		return nil, fmt.Errorf("create second Qizmo peer token: %w", err)
	}
	return newServerPeerHandshake(first, second), nil
}

func newPeerToken() (uint32, error) {
	var encoded [4]byte
	if _, err := rand.Read(encoded[:]); err != nil {
		return 0, err
	}
	token := binary.LittleEndian.Uint32(encoded[:]) & sequenceMask
	if token == 0 {
		token = 1
	}
	return token, nil
}

func newServerPeerHandshake(first, second uint32) *ServerPeerHandshake {
	advertisement := fmt.Sprintf("say proxy:teleport %d %d\n", first, second)
	responseText := fmt.Sprintf("say \"proxy:teleport %d %d\"", first, second)
	responseCommand := make([]byte, 0, len(responseText)+2)
	responseCommand = append(responseCommand, protocol.CLCStringCmd)
	responseCommand = append(responseCommand, responseText...)
	responseCommand = append(responseCommand, 0)
	return &ServerPeerHandshake{
		advertisement:   advertisement,
		responseCommand: responseCommand,
	}
}

func (h *ServerPeerHandshake) Advertise(packet []byte) ([]byte, bool) {
	if h.advertised {
		return packet, false
	}
	if _, ok := readRawServerHeader(packet); !ok {
		return packet, false
	}
	h.advertised = true

	command := make([]byte, 0, len(h.advertisement)+2)
	command = append(command, protocol.SVCStuffText)
	command = append(command, h.advertisement...)
	command = append(command, 0)

	out := make([]byte, 0, len(packet)+len(command))
	out = append(out, packet[:rawServerHeaderSize]...)
	out = append(out, command...)
	out = append(out, packet[rawServerHeaderSize:]...)
	return out, true
}

func (h *ServerPeerHandshake) ConsumeResponse(packet []byte) []byte {
	if !h.advertised || len(packet) < rawClientHeaderSize {
		return packet
	}

	operations, _, err := parseClientOperations(packet[rawClientHeaderSize:])
	if err != nil {
		return packet
	}
	offset := rawClientHeaderSize
	for _, operation := range operations {
		if bytes.Equal(operation.raw, h.responseCommand) {
			out := make([]byte, 0, len(packet)-len(operation.raw))
			out = append(out, packet[:offset]...)
			out = append(out, packet[offset+len(operation.raw):]...)
			return out
		}
		offset += len(operation.raw)
	}
	return packet
}
