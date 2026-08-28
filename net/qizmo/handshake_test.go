package qizmo

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/osm/quake/protocol"
)

func TestServerPeerHandshake(t *testing.T) {
	handshake := newServerPeerHandshake(1234, 5678)
	serverPacket := binary.LittleEndian.AppendUint32(nil, 10)
	serverPacket = binary.LittleEndian.AppendUint32(serverPacket, 9)
	serverPacket = append(serverPacket, protocol.SVCNOP)

	advertised, ok := handshake.Advertise(serverPacket)
	if !ok {
		t.Fatal("first sequenced packet was not advertised")
	}
	wantAdvertisement := append(
		[]byte{protocol.SVCStuffText},
		[]byte("say proxy:teleport 1234 5678\n\x00")...,
	)
	body := advertised[protocol.QWServerPacketHeaderSize:]
	if !bytes.Equal(body[:len(wantAdvertisement)], wantAdvertisement) {
		t.Fatalf("advertisement = %q, want %q", body, wantAdvertisement)
	}
	if !bytes.Equal(body[len(wantAdvertisement):], serverPacket[protocol.QWServerPacketHeaderSize:]) {
		t.Fatalf("server body changed: got %x, want %x", advertised, serverPacket)
	}
	got, ok := handshake.Advertise(serverPacket)
	if ok || !bytes.Equal(got, serverPacket) {
		t.Fatalf("second advertisement changed packet: got %x, want %x", got, serverPacket)
	}

	clientPacket := binary.LittleEndian.AppendUint32(nil, 11)
	clientPacket = binary.LittleEndian.AppendUint32(clientPacket, 10)
	clientPacket = binary.LittleEndian.AppendUint16(clientPacket, 27500)
	clientPacket = append(clientPacket, protocol.CLCNOP)
	clientPacket = append(clientPacket, handshake.responseCommand...)
	clientPacket = append(clientPacket, protocol.CLCDelta, 7)

	wantClient := append([]byte(nil), clientPacket[:rawClientHeaderSize+1]...)
	wantClient = append(wantClient, protocol.CLCDelta, 7)
	if got := handshake.ConsumeResponse(clientPacket); !bytes.Equal(got, wantClient) {
		t.Fatalf("consumed packet = %x, want %x", got, wantClient)
	}
	if got := handshake.ConsumeResponse(clientPacket); !bytes.Equal(got, wantClient) {
		t.Fatalf("retransmitted response = %x, want %x", got, wantClient)
	}
}

func TestServerPeerHandshakeLeavesConnectionlessPacketsAlone(t *testing.T) {
	handshake := newServerPeerHandshake(1234, 5678)
	packet := []byte("\xff\xff\xff\xffc123")
	got, ok := handshake.Advertise(packet)
	if ok || !bytes.Equal(got, packet) {
		t.Fatalf("connectionless packet changed: got %x, want %x", got, packet)
	}
}

func TestServerPeerHandshakeDoesNotConsumeUploadData(t *testing.T) {
	handshake := newServerPeerHandshake(1234, 5678)
	serverPacket := binary.LittleEndian.AppendUint32(nil, 1)
	serverPacket = binary.LittleEndian.AppendUint32(serverPacket, 0)
	handshake.Advertise(serverPacket)

	packet := binary.LittleEndian.AppendUint32(nil, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 0)
	packet = binary.LittleEndian.AppendUint16(packet, 27500)
	packet = append(packet, protocol.CLCUpload, byte(len(handshake.responseCommand)), 0, 0)
	packet = append(packet, handshake.responseCommand...)
	if got := handshake.ConsumeResponse(packet); !bytes.Equal(got, packet) {
		t.Fatalf("upload changed: got %x, want %x", got, packet)
	}
}
