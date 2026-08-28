package proxy

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	qizmonet "github.com/osm/quake/net/qizmo"
	"github.com/osm/quake/protocol"
	qizmocodec "github.com/osm/quake/qizmo"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/state"
)

func TestQizmoPeerWaitsForCompressedServerPacket(t *testing.T) {
	serverConn, peer := newQizmoPeerTestPair(t)
	defer serverConn.Close()
	defer peer.Close()

	seed := qizmoClientMovePacket(1, "new")
	warmup := qizmoClientMovePacket(2, "say waiting")
	for _, packet := range [][]byte{seed, warmup} {
		if err := peer.WritePacket(packet); err != nil {
			t.Fatalf("WritePacket: %v", err)
		}
	}

	buf := make([]byte, 512)
	var peerAddress *net.UDPAddr
	for i, want := range [][]byte{seed, warmup} {
		n, address, err := serverConn.ReadFromUDP(buf)
		if err != nil {
			t.Fatalf("ReadFromUDP raw packet %d: %v", i, err)
		}
		peerAddress = address
		if !bytes.Equal(buf[:n], want) {
			t.Fatalf("packet %d = %x, want raw %x", i, buf[:n], want)
		}
	}

	compressedServer, wantServer := testCompressedServerPacket(t)
	if _, err := serverConn.WriteToUDP(compressedServer, peerAddress); err != nil {
		t.Fatalf("WriteToUDP compressed server packet: %v", err)
	}
	n, err := peer.ReadPacket(buf)
	if err != nil {
		t.Fatalf("ReadPacket compressed server packet: %v", err)
	}
	if !bytes.Equal(buf[:n], wantServer) {
		t.Fatalf("decoded server packet = %x, want %x", buf[:n], wantServer)
	}

	compressedClient := qizmoClientPacket(3, "say ready")
	if err := peer.WritePacket(compressedClient); err != nil {
		t.Fatalf("WritePacket after compressed server packet: %v", err)
	}
	n, _, err = serverConn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("ReadFromUDP compressed client packet: %v", err)
	}
	if bytes.Equal(buf[:n], compressedClient) || n < 6 || buf[3]&0x40 == 0 {
		t.Fatalf("client packet was not compressed: %x", buf[:n])
	}
}

func TestQizmoPeerSharesPlayerNamesBetweenDirections(t *testing.T) {
	serverConn, peer := newQizmoPeerTestPair(t)
	defer serverConn.Close()
	defer peer.Close()

	if _, err := peer.serverDecoder.Decode(qizmoServerPlayerNamePacket(1, 5, "ToT_slime")); err != nil {
		t.Fatalf("observe server packet: %v", err)
	}
	seed := qizmoClientMovePacket(1, "new")
	if _, err := peer.clientEncoder.Encode(seed); err != nil {
		t.Fatalf("seed client encoder: %v", err)
	}
	peer.clientEncoder.Enable()
	packet := qizmoClientPacket(2, `say "ToT_slime ready"`)
	got, err := peer.clientEncoder.Encode(packet)
	if err != nil {
		t.Fatalf("encode client packet: %v", err)
	}

	encoder, err := qizmonet.NewClientEncoder(qizmonet.NewEndpointState())
	if err != nil {
		t.Fatalf("NewClientEncoder: %v", err)
	}
	if _, err := encoder.Encode(seed); err != nil {
		t.Fatalf("seed isolated encoder: %v", err)
	}
	encoder.Enable()
	withoutPlayerNames, err := encoder.Encode(packet)
	if err != nil {
		t.Fatalf("encode without player names: %v", err)
	}
	if bytes.Equal(got, withoutPlayerNames) {
		t.Fatal("client encoder did not use player names observed by the server decoder")
	}
}

func testCompressedServerPacket(t *testing.T) (compressed, decoded []byte) {
	t.Helper()
	frequencies, err := freq.NewTables(freq.DefaultCompressDat)
	if err != nil {
		t.Fatalf("load frequency tables: %v", err)
	}

	decoded = binary.LittleEndian.AppendUint32(nil, 1)
	decoded = binary.LittleEndian.AppendUint32(decoded, 1)
	decoded = append(decoded,
		protocol.SVCPlayerInfo, 0x00, 0x00, 0x00,
		0x34, 0x12, 0x78, 0x56, 0xbc, 0x9a, 0xde,
		protocol.SVCNOP,
	)
	encodeState := state.NewPacket(0)
	encodeState.SetCommandScale(1, 11)
	plan, ok := qizmocodec.NewLinkEncoder(frequencies, encodeState).PlanPacket(decoded, 0)
	if !ok {
		t.Fatal("server packet was not supported")
	}
	rangeEncoder := rangeenc.New()
	if err := plan.EncodeBody(rangeEncoder); err != nil {
		t.Fatalf("encode server packet: %v", err)
	}
	compressed = append([]byte{1, 0, 1, 0x40}, rangeEncoder.Finish()...)
	return compressed, decoded
}

func TestQizmoPeerConn(t *testing.T) {
	serverConn, peer := newQizmoPeerTestPair(t)
	defer serverConn.Close()
	defer peer.Close()

	wantFromProxy := []byte{0xff, 0xff, 0xff, 0xff, 'g'}
	wantFromQizmo := []byte{0xff, 0xff, 0xff, 0xff, 'c'}
	serverErr := make(chan error, 1)
	go func() {
		buf := make([]byte, 64)
		n, addr, err := serverConn.ReadFromUDP(buf)
		if err == nil && !bytes.Equal(buf[:n], wantFromProxy) {
			t.Errorf("packet from proxy = %x, want %x", buf[:n], wantFromProxy)
		}
		if err == nil {
			_, err = serverConn.WriteToUDP(wantFromQizmo, addr)
		}
		serverErr <- err
	}()

	if err := peer.WritePacket(wantFromProxy); err != nil {
		t.Fatalf("WritePacket: %v", err)
	}

	buf := make([]byte, 64)
	n, err := peer.ReadPacket(buf)
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if !bytes.Equal(buf[:n], wantFromQizmo) {
		t.Fatalf("packet from Qizmo = %x, want %x", buf[:n], wantFromQizmo)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("Qizmo side: %v", err)
	}
}

func newQizmoPeerTestPair(t *testing.T) (*net.UDPConn, *qizmoPeerConn) {
	t.Helper()
	serverConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatalf("ListenUDP: %v", err)
	}
	clientConn, err := net.DialUDP("udp", nil, serverConn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		serverConn.Close()
		t.Fatalf("DialUDP: %v", err)
	}
	peer, err := newQizmoPeerConn(clientConn)
	if err != nil {
		serverConn.Close()
		t.Fatalf("newQizmoPeerConn: %v", err)
	}
	return serverConn, peer
}

func qizmoClientPacket(sequence uint32, text string) []byte {
	packet := binary.LittleEndian.AppendUint32(nil, sequence)
	packet = binary.LittleEndian.AppendUint32(packet, 0)
	packet = binary.LittleEndian.AppendUint16(packet, 12345)
	packet = append(packet, protocol.CLCStringCmd)
	packet = append(packet, text...)
	return append(packet, 0)
}

func qizmoClientMovePacket(sequence uint32, text string) []byte {
	packet := binary.LittleEndian.AppendUint32(nil, sequence)
	packet = binary.LittleEndian.AppendUint32(packet, 0)
	packet = binary.LittleEndian.AppendUint16(packet, 12345)
	packet = append(packet,
		protocol.CLCMove, 0, 0, // checksum, lossage
		0, 10, // three unchanged usercmds and their msec values
		0, 11,
		0, 12,
		protocol.CLCStringCmd,
	)
	packet = append(packet, text...)
	return append(packet, 0)
}

func qizmoServerPlayerNamePacket(sequence uint32, slot byte, name string) []byte {
	packet := binary.LittleEndian.AppendUint32(nil, sequence)
	packet = binary.LittleEndian.AppendUint32(packet, 0)
	packet = append(packet, protocol.SVCUpdateUserInfo, slot)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet, `\name\`...)
	packet = append(packet, name...)
	return append(packet, 0)
}

func TestPeerAddress(t *testing.T) {
	tests := map[string]string{
		"localhost":       "localhost:27500",
		"localhost:28000": "localhost:28000",
		"[::1]":           "[::1]:27500",
	}
	for input, want := range tests {
		if got := peerAddress(input); got != want {
			t.Errorf("peerAddress(%q) = %q, want %q", input, got, want)
		}
	}
}
