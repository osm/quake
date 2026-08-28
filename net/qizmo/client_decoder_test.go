package qizmo

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/osm/quake/protocol"
)

const (
	qizmoMoveSeed1Hex = "4807000047070080f905030a00817d00bfc40d000d000d0547"
	qizmoMoveSeed2Hex = "4907000048070080f90503be00817d00bfc40d000d000d0548"
	qizmoPrivateHex   = "4a4749c7f905abdfbd7dcf4c7c56f98fda76bdac039fe0"
	qizmoMoveWireHex  = "4b474ac7f9058eff80"
	qizmoMovePlainHex = "4b0700004a070080f90503d600817d00bfc40d000d000d054a"

	qizmoMenuSeedHex  = "1503000014030000859103d30080b53e0d000d000d0514"
	qizmoMenuPlainHex = "160300801503000085910473617920222e6d656e752062696e647374642200030f0080b53e0d000d000d0515"
	qizmoMenuWireHex  = "16c315438591abdfb9848f77add30bf7276293805dfc"
)

func TestClientDecoderMatchesQizmoMoveCapture(t *testing.T) {
	decoder := newTestClientDecoder(t)

	// Consecutive packets captured on an original Qizmo-to-Qizmo link. The
	// first two establish move history. The next compressed packet contained
	// Qizmo-private strings that its peer consumed, so it is decoded only to
	// advance history. The final ordinary movement packet is compared with what
	// Qizmo 2.91 forwarded, with only its rewritten qport restored.
	for _, raw := range []string{qizmoMoveSeed1Hex, qizmoMoveSeed2Hex} {
		packet := mustDecodeHex(t, raw)
		got, err := decoder.Decode(packet)
		if err != nil {
			t.Fatalf("observe raw Qizmo packet: %v", err)
		}
		if !bytes.Equal(got, packet) {
			t.Fatalf("raw Qizmo packet changed: %x", got)
		}
	}

	privatePacket := mustDecodeHex(t, qizmoPrivateHex)
	if _, err := decoder.Decode(privatePacket); err != nil {
		t.Fatalf("decode original Qizmo private packet: %v", err)
	}

	wire := mustDecodeHex(t, qizmoMoveWireHex)
	want := mustDecodeHex(t, qizmoMovePlainHex)
	got, err := decoder.Decode(wire)
	if err != nil {
		t.Fatalf("decode original Qizmo packet: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("original Qizmo packet mismatch\ngot  %x\nwant %x", got, want)
	}
}

func TestClientDecoderMatchesQizmoMenuCapture(t *testing.T) {
	decoder := newTestClientDecoder(t)

	seed := mustDecodeHex(t, qizmoMenuSeedHex)
	if _, err := decoder.Decode(seed); err != nil {
		t.Fatalf("observe move-history seed: %v", err)
	}

	wire := mustDecodeHex(t, qizmoMenuWireHex)
	want := mustDecodeHex(t, qizmoMenuPlainHex)
	got, err := decoder.Decode(wire)
	if err != nil {
		t.Fatalf("decode original Qizmo dictionary packet: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("original Qizmo dictionary packet mismatch\ngot  %x\nwant %x", got, want)
	}
}

func TestStaticClientStringTokens(t *testing.T) {
	tests := []struct {
		token byte
		want  string
		ok    bool
	}{
		{token: 31},
		{token: 32, want: "bottomcolor", ok: true},
		{token: 54, want: "say \"", ok: true},
		{token: 58, want: "setinfo \"", ok: true},
		{token: 59},
		{token: 64, want: "lost ", ok: true},
		{token: 255, want: "  0.8", ok: true},
	}
	for _, test := range tests {
		got, ok := staticClientString(test.token)
		if got != test.want || ok != test.ok {
			t.Errorf("token %d = %q, %v; want %q, %v", test.token, got, ok, test.want, test.ok)
		}
	}
}

func TestClientCodecRoundTrip(t *testing.T) {
	encoder := newTestClientEncoder(t)
	decoder := newTestClientDecoder(t)

	seed := clientPacket(1, 0, 0x1234, clientMoveBody(0x10, 5, 10, 11, 12)...)
	requireClientCodecRoundTrip(t, encoder, decoder, seed, false)
	encoder.Enable()

	packets := [][]byte{
		clientPacket(2, 1, 0x1234, clientMoveBody(0xab, 7, 11, 12, 13)...),
		clientPacket(
			3, 2, 0x1234,
			append(clientMoveBody(0xac, 8, 12, 13, 14),
				protocol.CLCDelta, 0x7e,
				protocol.CLCTMove, 1, 2, 3, 4, 5, 6,
				protocol.CLCUpload, 3, 0, 50, 0x10, 0x12, 0x11,
			)...,
		),
		clientPacket(
			0x80000004, 0x80000003, 0x1234,
			protocol.CLCStringCmd, 's', 'a', 'y', ' ', 'h', 'i', 0,
			protocol.CLCDelta, 3,
		),
	}
	for _, packet := range packets {
		requireClientCodecRoundTrip(t, encoder, decoder, packet, true)
	}
}

func TestClientCodecRoundTripMoveFields(t *testing.T) {
	encoder := newTestClientEncoder(t)
	decoder := newTestClientDecoder(t)

	seed := clientPacket(20, 0, 1, clientMovePitchBody(0x10, 0x1000)...)
	requireClientCodecRoundTrip(t, encoder, decoder, seed, false)
	encoder.Enable()

	move := clientMove{checksum: 0x7a, lossage: 9}
	move.commands[0] = clientCommand{angle: [3]uint16{0x1000, 0, 0}, msec: 13}
	move.commands[1] = clientCommand{angle: [3]uint16{0x1000, 0, 0}, msec: 13}
	move.commands[2] = clientCommand{
		angle:   [3]uint16{0x1202, 0x1f00, 0x3011},
		move:    [3]uint16{300, 0xff38, 0},
		buttons: 5,
		impulse: 2,
		msec:    15,
	}
	body := appendClientOperation(nil, clientOperation{opcode: protocol.CLCMove, move: &move})
	packet := clientPacket(21, 20, 1, body...)
	requireClientCodecRoundTrip(t, encoder, decoder, packet, true)
}

func TestClientDecoderTracksRawFallback(t *testing.T) {
	encoder := newTestClientEncoder(t)
	decoder := newTestClientDecoder(t)

	seed := clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...)
	requireClientCodecRoundTrip(t, encoder, decoder, seed, false)
	encoder.Enable()

	unsupported := clientPacket(
		2, 1, 1,
		append(clientMoveBody(0x11, 0, 11, 12, 13), protocol.CLCNOP)...,
	)
	requireClientCodecRoundTrip(t, encoder, decoder, unsupported, false)

	next := clientPacket(3, 2, 1, clientMoveBody(0x12, 0, 12, 13, 14)...)
	requireClientCodecRoundTrip(t, encoder, decoder, next, true)
}

func requireClientCodecRoundTrip(
	t *testing.T,
	encoder *ClientEncoder,
	decoder *ClientDecoder,
	packet []byte,
	wantCompressed bool,
) {
	t.Helper()
	encoded, err := encoder.Encode(packet)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	compressed := !bytes.Equal(encoded, packet)
	if compressed != wantCompressed {
		t.Fatalf("compressed = %v, want %v\nencoded %x\npacket  %x", compressed, wantCompressed, encoded, packet)
	}
	got, err := decoder.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(got, packet) {
		t.Fatalf("client codec mismatch\ngot  %x\nwant %x\nwire %x", got, packet, encoded)
	}
}

func mustDecodeHex(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatalf("decode test hex: %v", err)
	}
	return decoded
}

func newTestClientDecoder(t *testing.T) *ClientDecoder {
	t.Helper()
	decoder, err := NewClientDecoder(NewEndpointState())
	if err != nil {
		t.Fatalf("NewClientDecoder: %v", err)
	}
	return decoder
}
