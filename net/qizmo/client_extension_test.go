package qizmo

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
)

func TestClientCodecQizmoExtensions(t *testing.T) {
	peerAssociation := sequentialBytes(qizmoprotocol.CLCPeerAssociationPayloadSize, 0x10)
	voiceStart := []byte{'t', 'e', 'a', 'm', ' ', '7', ' ', '8', '3', ' ', 0xff, 0}
	voiceStop := []byte("7\x00")
	voiceGSM := sequentialBytes(qizmoprotocol.CLCVoiceGSMPayloadSize, 0x40)
	body := append([]byte{qizmoprotocol.CLCPeerAssociation}, peerAssociation...)
	body = append(body, qizmoprotocol.CLCVoiceStart)
	body = append(body, voiceStart...)
	body = append(body, qizmoprotocol.CLCVoiceStop)
	body = append(body, voiceStop...)
	body = append(body, qizmoprotocol.CLCVoiceGSM)
	body = append(body, voiceGSM...)
	body = append(body, qizmoprotocol.CLCRequestS2CResync)
	body = append(body, protocol.CLCDelta, 1)

	seed := clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...)
	packet := clientPacket(2, 1, 1, body...)
	encoder := newTestClientEncoder(t)
	encodeClientPacket(t, encoder, seed)
	encoder.Enable()
	encoded := encodeClientPacket(t, encoder, packet)
	if bytes.Equal(encoded, packet) {
		t.Fatal("Qizmo extensions remained raw")
	}

	t.Run("wire format", func(t *testing.T) {
		reader := testBitReader{data: encoded[compressedClientHeaderSize:]}
		requireSymbol(t, encoder, &reader, freq.CLCSequenceDelta, 1)
		requireLiteralExtension(t, encoder, &reader, qizmoprotocol.CLCPeerAssociation, peerAssociation)
		requireStringExtension(t, encoder, &reader, qizmoprotocol.CLCVoiceStart, voiceStart)
		requireStringExtension(t, encoder, &reader, qizmoprotocol.CLCVoiceStop, voiceStop)
		requireLiteralExtension(t, encoder, &reader, qizmoprotocol.CLCVoiceGSM, voiceGSM)
		requireSymbol(t, encoder, &reader, freq.CLCType, qizmoprotocol.CLCRequestS2CResync)
		requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCBad)
	})

	t.Run("Qizmo peer", func(t *testing.T) {
		got, err := seededClientDecoder(t, seed).Decode(encoded)
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if !bytes.Equal(got, packet) {
			t.Fatalf("decoded extensions mismatch\ngot  %x\nwant %x", got, packet)
		}
	})

	t.Run("ordinary server", func(t *testing.T) {
		ordinary, extensions, err := seededClientDecoder(t, seed).DecodeWithExtensions(encoded)
		if err != nil {
			t.Fatalf("DecodeWithExtensions: %v", err)
		}
		wantOrdinary := append(append([]byte(nil), packet[:rawClientHeaderSize]...), protocol.CLCDelta, 1)
		if !bytes.Equal(ordinary, wantOrdinary) {
			t.Fatalf("ordinary packet = %x, want %x", ordinary, wantOrdinary)
		}
		wantExtensions := []ClientExtension{
			{Opcode: qizmoprotocol.CLCPeerAssociation, Payload: peerAssociation},
			{Opcode: qizmoprotocol.CLCVoiceStart, Payload: voiceStart},
			{Opcode: qizmoprotocol.CLCVoiceStop, Payload: voiceStop},
			{Opcode: qizmoprotocol.CLCVoiceGSM, Payload: voiceGSM},
			{Opcode: qizmoprotocol.CLCRequestS2CResync},
		}
		if !reflect.DeepEqual(extensions, wantExtensions) {
			t.Fatalf("extensions = %#v, want %#v", extensions, wantExtensions)
		}
	})
}

func TestClientEncoderLeavesRawVoiceUncompressed(t *testing.T) {
	seed := clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...)
	voice := sequentialBytes(qizmoprotocol.CLCVoiceRawPayloadSize, 0x20)
	body := append([]byte{qizmoprotocol.CLCVoiceRaw}, voice...)
	move := clientMoveBody(0x11, 0, 11, 12, 13)
	body = append(body, move...)
	packet := clientPacket(2, 1, 1, body...)

	encoder := newTestClientEncoder(t)
	encodeClientPacket(t, encoder, seed)
	encoder.Enable()
	if got := encodeClientPacket(t, encoder, packet); !bytes.Equal(got, packet) {
		t.Fatalf("raw voice packet was compressed: %x", got)
	}
	next := clientPacket(3, 2, 1, protocol.CLCStringCmd, 'x', 0)
	if got := encodeClientPacket(t, encoder, next); bytes.Equal(got, next) {
		t.Fatal("raw voice packet did not preserve following move history")
	}

	decoder := newTestClientDecoder(t)
	ordinary, extensions, err := decoder.DecodeWithExtensions(packet)
	if err != nil {
		t.Fatalf("DecodeWithExtensions: %v", err)
	}
	wantOrdinary := append(append([]byte(nil), packet[:rawClientHeaderSize]...), move...)
	if !bytes.Equal(ordinary, wantOrdinary) {
		t.Fatalf("ordinary packet = %x, want %x", ordinary, wantOrdinary)
	}
	want := []ClientExtension{{Opcode: qizmoprotocol.CLCVoiceRaw, Payload: voice}}
	if !reflect.DeepEqual(extensions, want) {
		t.Fatalf("extensions = %#v, want %#v", extensions, want)
	}
}

func TestParseClientExtensionsRejectsTruncatedPayloads(t *testing.T) {
	tests := []struct {
		name string
		body []byte
	}{
		{"peer association", append([]byte{qizmoprotocol.CLCPeerAssociation}, make([]byte, qizmoprotocol.CLCPeerAssociationPayloadSize-1)...)},
		{"voice start", []byte{qizmoprotocol.CLCVoiceStart, 'x'}},
		{"voice stop", []byte{qizmoprotocol.CLCVoiceStop, '1'}},
		{"raw voice", append([]byte{qizmoprotocol.CLCVoiceRaw}, make([]byte, qizmoprotocol.CLCVoiceRawPayloadSize-1)...)},
		{"GSM voice", append([]byte{qizmoprotocol.CLCVoiceGSM}, make([]byte, qizmoprotocol.CLCVoiceGSMPayloadSize-1)...)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := parseClientOperations(test.body); err == nil {
				t.Fatal("truncated extension was accepted")
			}
		})
	}
}

func TestClientDecoderRejectsTruncatedCompressedExtensions(t *testing.T) {
	tests := []struct {
		name   string
		opcode byte
		size   int
	}{
		{"peer association", qizmoprotocol.CLCPeerAssociation, qizmoprotocol.CLCPeerAssociationPayloadSize},
		{"GSM voice", qizmoprotocol.CLCVoiceGSM, qizmoprotocol.CLCVoiceGSMPayloadSize},
	}
	seed := clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoder := newTestClientEncoder(t)
			writer := &bitWriter{}
			if err := encoder.writeSymbol(writer, freq.CLCSequenceDelta, 1); err != nil {
				t.Fatalf("encode sequence delta: %v", err)
			}
			if err := encoder.writeSymbol(writer, freq.CLCType, test.opcode); err != nil {
				t.Fatalf("encode opcode: %v", err)
			}
			for range test.size - 1 {
				writer.writeByte(0)
			}
			wire := append([]byte{2, 0, 1, 0x40, 1, 0}, writer.data...)
			if _, err := seededClientDecoder(t, seed).Decode(wire); err == nil {
				t.Fatal("truncated compressed extension was accepted")
			}
		})
	}
}

func TestClientDecoderRejectsCompressedRawVoice(t *testing.T) {
	encoder := newTestClientEncoder(t)
	writer := &bitWriter{}
	if err := encoder.writeSymbol(writer, freq.CLCSequenceDelta, 1); err != nil {
		t.Fatalf("encode sequence delta: %v", err)
	}
	if err := encoder.writeSymbol(writer, freq.CLCType, qizmoprotocol.CLCVoiceRaw); err != nil {
		t.Fatalf("encode opcode: %v", err)
	}
	for range qizmoprotocol.CLCVoiceRawPayloadSize {
		writer.writeByte(0)
	}
	wire := append([]byte{2, 0, 1, 0x40, 1, 0}, writer.data...)
	seed := clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...)
	if _, err := seededClientDecoder(t, seed).Decode(wire); err == nil {
		t.Fatal("compressed raw voice frame was accepted")
	}
}

func seededClientDecoder(t *testing.T, seed []byte) *ClientDecoder {
	t.Helper()
	decoder := newTestClientDecoder(t)
	if _, err := decoder.Decode(seed); err != nil {
		t.Fatalf("seed ClientDecoder: %v", err)
	}
	return decoder
}

func sequentialBytes(size int, first byte) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = first + byte(i)
	}
	return data
}

func requireLiteralExtension(
	t *testing.T,
	encoder *ClientEncoder,
	reader *testBitReader,
	opcode byte,
	payload []byte,
) {
	t.Helper()
	requireSymbol(t, encoder, reader, freq.CLCType, opcode)
	for _, value := range payload {
		requireByte(t, reader, value)
	}
}

func requireStringExtension(
	t *testing.T,
	encoder *ClientEncoder,
	reader *testBitReader,
	opcode byte,
	payload []byte,
) {
	t.Helper()
	requireSymbol(t, encoder, reader, freq.CLCType, opcode)
	for _, value := range payload {
		requireSymbol(t, encoder, reader, freq.CLCStringByte, value)
	}
}
