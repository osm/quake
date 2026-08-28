package qizmo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
)

func TestDynamicClientStringTokens(t *testing.T) {
	state := NewEndpointState()
	encoder, err := NewClientEncoder(state)
	if err != nil {
		t.Fatalf("NewClientEncoder: %v", err)
	}
	decoder, err := NewClientDecoder(state)
	if err != nil {
		t.Fatalf("NewClientDecoder: %v", err)
	}

	for slot := range clientDynamicTokenCount {
		clear(state.playerNames[:])
		name := fmt.Sprintf("~player-%02d~", slot)
		state.playerNames[slot] = name

		writer := &bitWriter{}
		if err := encoder.encodeClientString(writer, []byte(name+"\x00")); err != nil {
			t.Fatalf("slot %d encode: %v", slot, err)
		}

		encoded := testBitReader{data: writer.data}
		requireSymbol(t, encoder, &encoded, freq.CLCStringByte, clientStringEscape)
		requireSymbol(t, encoder, &encoded, freq.CLCStringToken, byte(slot))
		requireSymbol(t, encoder, &encoded, freq.CLCStringByte, 0)

		got, err := decoder.decodeClientString(&bitReader{data: writer.data})
		if err != nil {
			t.Fatalf("slot %d decode: %v", slot, err)
		}
		if want := []byte(name + "\x00"); !bytes.Equal(got, want) {
			t.Fatalf("slot %d decoded %q, want %q", slot, got, want)
		}
	}
}

func TestOriginalQizmoDynamicClientStringToken(t *testing.T) {
	state := NewEndpointState()
	state.playerNames[0] = "unnamed"
	decoder, err := NewClientDecoder(state)
	if err != nil {
		t.Fatalf("NewClientDecoder: %v", err)
	}

	// Original Qizmo 2.91 encoded `say "unnamed ready"` after learning that
	// player slot 0 was named "unnamed". The remaining bits contain the move
	// command that shared this captured packet.
	wire := mustDecodeHex(t, "7ec67dc61ff9abdfbbde1e97da13db0081f8")
	stringReader := func() *bitReader {
		reader := &bitReader{data: wire[compressedClientHeaderSize:]}
		requireDecodedClientSymbol(t, decoder, reader, freq.CLCSequenceDelta, 1)
		requireDecodedClientSymbol(t, decoder, reader, freq.CLCType, protocol.CLCStringCmd)
		return reader
	}

	reader := stringReader()
	requireDecodedClientSymbol(t, decoder, reader, freq.CLCStringByte, clientStringEscape)
	requireDecodedClientSymbol(t, decoder, reader, freq.CLCStringToken, 54) // `say "`
	requireDecodedClientSymbol(t, decoder, reader, freq.CLCStringByte, clientStringEscape)
	requireDecodedClientSymbol(t, decoder, reader, freq.CLCStringToken, 0) // player slot 0

	got, err := decoder.decodeClientString(stringReader())
	if err != nil {
		t.Fatalf("decode captured string: %v", err)
	}
	if want := []byte("say \"unnamed ready\"\x00"); !bytes.Equal(got, want) {
		t.Fatalf("captured string = %q, want %q", got, want)
	}

	encoder, err := NewClientEncoder(state)
	if err != nil {
		t.Fatalf("NewClientEncoder: %v", err)
	}
	written := &bitWriter{}
	if err := encoder.writeSymbol(written, freq.CLCSequenceDelta, 1); err != nil {
		t.Fatalf("encode sequence delta: %v", err)
	}
	if err := encoder.writeSymbol(written, freq.CLCType, protocol.CLCStringCmd); err != nil {
		t.Fatalf("encode string opcode: %v", err)
	}
	if err := encoder.encodeClientString(written, []byte("say \"unnamed ready\"\x00")); err != nil {
		t.Fatalf("encode captured string: %v", err)
	}
	requireBitPrefix(t, wire[compressedClientHeaderSize:], written)
}

func TestClientStringDictionaryPriority(t *testing.T) {
	var names [clientDynamicTokenCount]string

	// The proxy dictionary wins even if a player has the same name.
	names[7] = `say "`
	if token, _, ok := matchClientString([]byte(`say "hello`), &names); !ok || token != 54 {
		t.Fatalf("proxy match = token %d, %v; want token 54", token, ok)
	}

	// Player names are considered before the common teamplay dictionary.
	names[7] = "lost "
	if token, _, ok := matchClientString([]byte("lost powerup"), &names); !ok || token != 7 {
		t.Fatalf("dynamic match = token %d, %v; want token 7", token, ok)
	}

	// Within the dynamic dictionary Qizmo chooses the longest name.
	names[2] = "ToT"
	names[5] = "ToT_slime"
	if token, size, ok := matchClientString([]byte("ToT_slime: ready"), &names); !ok || token != 5 || size != len("ToT_slime") {
		t.Fatalf("longest dynamic match = token %d, size %d, %v; want token 5, size 9", token, size, ok)
	}
}

func TestDynamicClientStringTokenRequiresKnownPlayer(t *testing.T) {
	encoder := newTestClientEncoder(t)
	decoder := newTestClientDecoder(t)

	writer := &bitWriter{}
	if err := encoder.writeSymbol(writer, freq.CLCStringByte, clientStringEscape); err != nil {
		t.Fatalf("encode escape: %v", err)
	}
	if err := encoder.writeSymbol(writer, freq.CLCStringToken, 5); err != nil {
		t.Fatalf("encode token: %v", err)
	}
	if _, err := decoder.decodeClientString(&bitReader{data: writer.data}); err == nil {
		t.Fatal("unknown dynamic token decoded without an error")
	}
}

func TestClientCodecRoundTripDynamicPlayerName(t *testing.T) {
	encodeState := NewEndpointState()
	encoder, err := NewClientEncoder(encodeState)
	if err != nil {
		t.Fatalf("NewClientEncoder: %v", err)
	}
	serverDecoder, err := NewServerDecoder(encodeState)
	if err != nil {
		t.Fatalf("NewServerDecoder: %v", err)
	}

	decodeState := NewEndpointState()
	decoder, err := NewClientDecoder(decodeState)
	if err != nil {
		t.Fatalf("NewClientDecoder: %v", err)
	}
	serverEncoder, err := NewServerEncoder(decodeState)
	if err != nil {
		t.Fatalf("NewServerEncoder: %v", err)
	}

	identity := serverPlayerNamePacket(1, 5, "ToT_slime")
	if _, err := serverDecoder.Decode(identity); err != nil {
		t.Fatalf("client endpoint observe player name: %v", err)
	}
	if _, err := serverEncoder.Encode(identity); err != nil {
		t.Fatalf("server endpoint observe player name: %v", err)
	}
	if got := encodeState.playerNames[5]; got != "ToT_slime" {
		t.Fatalf("client endpoint player name = %q, want ToT_slime", got)
	}
	if got := decodeState.playerNames[5]; got != "ToT_slime" {
		t.Fatalf("server endpoint player name = %q, want ToT_slime", got)
	}

	seed := clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...)
	requireClientCodecRoundTrip(t, encoder, decoder, seed, false)
	encoder.Enable()

	packet := clientPacket(
		2, 1, 1,
		protocol.CLCStringCmd,
		's', 'a', 'y', ' ', '"',
		'T', 'o', 'T', '_', 's', 'l', 'i', 'm', 'e',
		':', ' ', 'r', 'e', 'a', 'd', 'y', '"', 0,
	)
	requireClientCodecRoundTrip(t, encoder, decoder, packet, true)
}

func serverPlayerNamePacket(sequence uint32, slot byte, name string) []byte {
	packet := binary.LittleEndian.AppendUint32(nil, sequence)
	packet = binary.LittleEndian.AppendUint32(packet, 0)
	packet = append(packet, protocol.SVCUpdateUserInfo, slot)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet, `\name\`...)
	packet = append(packet, name...)
	return append(packet, 0)
}

func requireDecodedClientSymbol(
	t *testing.T,
	decoder *ClientDecoder,
	reader *bitReader,
	model uint32,
	want byte,
) {
	t.Helper()
	got, err := decoder.readSymbol(reader, model)
	if err != nil {
		t.Fatalf("decode model %#x: %v", model, err)
	}
	if got != want {
		t.Fatalf("model %#x symbol = %#x, want %#x", model, got, want)
	}
}

func requireBitPrefix(t *testing.T, want []byte, got *bitWriter) {
	t.Helper()
	for bit := range got.bits {
		wantBit := want[bit/8] >> (7 - (bit & 7)) & 1
		gotBit := got.data[bit/8] >> (7 - (bit & 7)) & 1
		if gotBit != wantBit {
			t.Fatalf("encoded bit %d = %d, want %d", bit, gotBit, wantBit)
		}
	}
}
