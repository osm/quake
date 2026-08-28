package qizmo

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
)

func TestClientEncoderStringAndDelta(t *testing.T) {
	encoder := newTestClientEncoder(t)
	seed := clientPacket(100, 1, 0x1234, clientMoveBody(0x10, 0, 10, 11, 12)...)
	if got := encodeClientPacket(t, encoder, seed); !bytes.Equal(got, seed) {
		t.Fatalf("disabled encoder changed packet: got %x, want %x", got, seed)
	}

	encoder.Enable()
	packet := clientPacket(
		0x80000066,
		0x8000002a,
		0x1234,
		protocol.CLCStringCmd, 's', 'a', 'y', ' ', 'h', 'i', 0,
		protocol.CLCDelta, 0x2a,
	)
	got := encodeClientPacket(t, encoder, packet)
	wantHeader := []byte{0x66, 0xc0, 0x2a, 0xc0, 0x34, 0x12}
	if len(got) <= len(wantHeader) || !bytes.Equal(got[:len(wantHeader)], wantHeader) {
		t.Fatalf("compressed header = %x, want prefix %x", got, wantHeader)
	}

	reader := testBitReader{data: got[len(wantHeader):]}
	requireSymbol(t, encoder, &reader, freq.CLCSequenceDelta, 2)
	requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCStringCmd)
	requireSymbol(t, encoder, &reader, freq.CLCStringByte, clientStringEscape)
	requireSymbol(t, encoder, &reader, freq.CLCStringToken, 52) // "say "
	for _, value := range []byte("hi\x00") {
		requireSymbol(t, encoder, &reader, freq.CLCStringByte, value)
	}
	// The matching delta command is carried by bit 6 of the sequence header,
	// so it does not appear in the command stream.
	requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCBad)
}

func TestClientEncoderMatchesQizmoMenuCapture(t *testing.T) {
	encoder := newTestClientEncoder(t)
	seed := mustDecodeHex(t, qizmoMenuSeedHex)
	if got := encodeClientPacket(t, encoder, seed); !bytes.Equal(got, seed) {
		t.Fatalf("move-history seed changed: got %x", got)
	}
	encoder.Enable()

	packet := mustDecodeHex(t, qizmoMenuPlainHex)
	want := mustDecodeHex(t, qizmoMenuWireHex)
	if got := encodeClientPacket(t, encoder, packet); !bytes.Equal(got, want) {
		t.Fatalf("original Qizmo dictionary packet mismatch\ngot  %x\nwant %x", got, want)
	}
}

func TestClientEncoderFallsBackWithoutLosingSequenceState(t *testing.T) {
	encoder := newTestClientEncoder(t)
	encodeClientPacket(t, encoder, clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...))
	encoder.Enable()

	unsupportedBody := append(clientMoveBody(0x11, 0, 11, 12, 13), protocol.CLCNOP)
	unsupported := clientPacket(2, 1, 1, unsupportedBody...)
	if got := encodeClientPacket(t, encoder, unsupported); !bytes.Equal(got, unsupported) {
		t.Fatalf("unsupported packet was compressed: got %x", got)
	}

	got := encodeClientPacket(t, encoder, clientPacket(3, 2, 1, protocol.CLCStringCmd, 'b', 0))
	if bytes.Equal(got, clientPacket(3, 2, 1, protocol.CLCStringCmd, 'b', 0)) {
		t.Fatal("supported packet after raw fallback was not compressed")
	}
	reader := testBitReader{data: got[compressedClientHeaderSize:]}
	requireSymbol(t, encoder, &reader, freq.CLCSequenceDelta, 1)
}

func TestClientEncoderDoesNotAdvanceOnError(t *testing.T) {
	encoder, err := newClientEncoder(make([]byte, len(freq.DefaultCompressDat)), NewEndpointState())
	if err != nil {
		t.Fatalf("newClientEncoder: %v", err)
	}
	seed := clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...)
	encodeClientPacket(t, encoder, seed)
	encoder.Enable()

	wantSequences := encoder.sequences
	wantMoves := encoder.moves

	packet := clientPacket(2, 1, 1, protocol.CLCStringCmd, 'x', 0)
	if _, err := encoder.Encode(packet); err == nil {
		t.Fatal("Encode succeeded with empty Huffman tables")
	}
	if encoder.sequences != wantSequences || encoder.moves != wantMoves {
		t.Fatal("encoder state changed after failed packet")
	}
}

func TestClientEncoderMovePrediction(t *testing.T) {
	encoder := newTestClientEncoder(t)
	seed := clientPacket(1, 0, 1, clientMoveBody(0x10, 5, 10, 11, 12)...)
	encodeClientPacket(t, encoder, seed)
	encoder.Enable()

	packet := clientPacket(2, 1, 1, clientMoveBody(0xab, 7, 11, 12, 13)...)
	got := encodeClientPacket(t, encoder, packet)
	if bytes.Equal(got, packet) {
		t.Fatal("move packet was not compressed")
	}

	reader := testBitReader{data: got[compressedClientHeaderSize:]}
	requireSymbol(t, encoder, &reader, freq.CLCSequenceDelta, 1)
	requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCMove)
	requireSymbol(t, encoder, &reader, freq.CLCMoveChecksum, 0xab)
	requireSymbol(t, encoder, &reader, freq.CLCMoveLossageDelta, 2)
	requireSymbol(t, encoder, &reader, freq.CLCMoveMaskLoXOR, 0)
	requireSymbol(t, encoder, &reader, freq.CLCMoveMaskHiXOR, 0x20)
	requireSymbol(t, encoder, &reader, freq.CLCMoveMsecDelta, 1)
	requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCBad)
}

func TestClientEncoderKeepsUntransmittedRedundantMoveHistory(t *testing.T) {
	encoder := newTestClientEncoder(t)
	const oldPitch = 0x1234
	seed := clientPacket(11, 0, 1, clientMovePitchBody(0x10, oldPitch)...)
	encodeClientPacket(t, encoder, seed)
	encoder.Enable()

	// A client may reset its redundant command window while changing servers.
	// With a sequence delta of one, Qizmo receives only command 12; commands
	// 10 and 11 in this packet must not replace its existing history.
	reset := clientPacket(12, 0, 1, clientMovePitchBody(0x11, 0)...)
	if got := encodeClientPacket(t, encoder, reset); bytes.Equal(got, reset) {
		t.Fatal("reset packet was not compressed")
	}

	previous, ok := clientMoveRecordAt(&encoder.moves, 11)
	if !ok {
		t.Fatal("command 11 is missing from encoder history")
	}
	if got := previous.command.angle[0]; got != oldPitch {
		t.Fatalf("command 11 pitch = %#x, want retained %#x", got, oldPitch)
	}
	current, ok := clientMoveRecordAt(&encoder.moves, 12)
	if !ok || current.command.angle[0] != 0 {
		t.Fatalf("command 12 = %+v, want reset pitch", current)
	}
}

func TestClientMovePredictionUsesQizmoAngleOrder(t *testing.T) {
	var yawCommand clientCommand
	yawCommand.angle[1] = 0x1234
	yawPredictor := newClientMovePredictor(clientCommand{}, clientCommand{}, 0)
	yawDelta := yawPredictor.plan(yawCommand)
	if yawDelta.mask != 0x0003 {
		t.Fatalf("yaw mask = %#04x, want %#04x", yawDelta.mask, uint16(0x0003))
	}

	var pitchCommand clientCommand
	pitchCommand.angle[0] = 0x1234
	pitchPredictor := newClientMovePredictor(clientCommand{}, clientCommand{}, 0)
	pitchDelta := pitchPredictor.plan(pitchCommand)
	if pitchDelta.mask != 0x000c {
		t.Fatalf("pitch mask = %#04x, want %#04x", pitchDelta.mask, uint16(0x000c))
	}

	encoder := newTestClientEncoder(t)
	writer := &bitWriter{}
	if err := encoder.encodeClientCommandDelta(writer, yawDelta, 0); err != nil {
		t.Fatalf("encode yaw delta: %v", err)
	}
	reader := testBitReader{data: writer.data}
	requireSymbol(t, encoder, &reader, freq.CLCMoveMaskLoXOR, 0x03)
	requireSymbol(t, encoder, &reader, freq.CLCMoveMaskHiXOR, 0x00)
	requireSymbol(t, encoder, &reader, freq.CLCMoveYawDeltaLo, 0x34)
	requireSymbol(t, encoder, &reader, freq.CLCMoveYawDeltaHi, 0x12)
}

func TestClientEncoderOtherSupportedCommands(t *testing.T) {
	encoder := newTestClientEncoder(t)
	encodeClientPacket(t, encoder, clientPacket(1, 0, 1, clientMoveBody(0x10, 0, 10, 11, 12)...))
	encoder.Enable()

	packet := clientPacket(
		2, 1, 1,
		protocol.CLCDelta, 0x7e,
		protocol.CLCTMove, 1, 2, 3, 4, 5, 6,
		protocol.CLCUpload, 3, 0, 50, 0x10, 0x12, 0x11,
	)
	got := encodeClientPacket(t, encoder, packet)
	reader := testBitReader{data: got[compressedClientHeaderSize:]}
	requireSymbol(t, encoder, &reader, freq.CLCSequenceDelta, 1)

	requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCDelta)
	requireByte(t, &reader, 0x7e)

	requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCTMove)
	for value := byte(1); value <= 6; value++ {
		requireByte(t, &reader, value)
	}

	requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCUpload)
	for _, value := range []byte{3, 0, 50} {
		requireByte(t, &reader, value)
	}
	for _, delta := range []byte{0x10, 0x02, 0xff} {
		requireSymbol(t, encoder, &reader, freq.CLCUploadDataByteDelta, delta)
	}
	requireSymbol(t, encoder, &reader, freq.CLCType, protocol.CLCBad)
}

func TestClientEncoderWaitsForMoveHistory(t *testing.T) {
	encoder := newTestClientEncoder(t)
	encoder.Enable()

	first := clientPacket(1, 0, 1, protocol.CLCStringCmd, 'a', 0)
	second := clientPacket(2, 1, 1, protocol.CLCStringCmd, 'b', 0)
	for _, packet := range [][]byte{first, second} {
		if got := encodeClientPacket(t, encoder, packet); !bytes.Equal(got, packet) {
			t.Fatalf("string-only packet compressed without move history: got %x", got)
		}
	}

	seed := clientPacket(3, 2, 1, clientMoveBody(0x10, 0, 10, 11, 12)...)
	if got := encodeClientPacket(t, encoder, seed); !bytes.Equal(got, seed) {
		t.Fatalf("first move packet compressed without prior history: got %x", got)
	}

	packet := clientPacket(4, 3, 1, protocol.CLCStringCmd, 'c', 0)
	if got := encodeClientPacket(t, encoder, packet); bytes.Equal(got, packet) {
		t.Fatal("packet after move-history seed was not compressed")
	}
}

func newTestClientEncoder(t *testing.T) *ClientEncoder {
	t.Helper()
	encoder, err := NewClientEncoder(NewEndpointState())
	if err != nil {
		t.Fatalf("NewClientEncoder: %v", err)
	}
	return encoder
}

func encodeClientPacket(t *testing.T, encoder *ClientEncoder, packet []byte) []byte {
	t.Helper()
	got, err := encoder.Encode(packet)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return got
}

func clientPacket(sequence, ack uint32, qport uint16, body ...byte) []byte {
	packet := binary.LittleEndian.AppendUint32(nil, sequence)
	packet = binary.LittleEndian.AppendUint32(packet, ack)
	packet = binary.LittleEndian.AppendUint16(packet, qport)
	return append(packet, body...)
}

func clientMoveBody(checksum, lossage, firstMsec, secondMsec, thirdMsec byte) []byte {
	return []byte{
		protocol.CLCMove, checksum, lossage,
		0, firstMsec,
		0, secondMsec,
		0, thirdMsec,
	}
}

func clientMovePitchBody(checksum byte, pitch uint16) []byte {
	return []byte{
		protocol.CLCMove, checksum, 0,
		protocol.CMAngle1, byte(pitch), byte(pitch >> 8), 13,
		0, 13,
		0, 13,
	}
}

type testBitReader struct {
	data []byte
	bits int
}

func (r *testBitReader) readBit(t *testing.T) uint32 {
	t.Helper()
	if r.bits >= len(r.data)*8 {
		t.Fatal("unexpected end of Huffman stream")
	}
	bit := uint32(r.data[r.bits/8] >> (7 - (r.bits & 7)) & 1)
	r.bits++
	return bit
}

func requireByte(t *testing.T, reader *testBitReader, want byte) {
	t.Helper()
	var got byte
	for range 8 {
		got = got<<1 | byte(reader.readBit(t))
	}
	if got != want {
		t.Fatalf("raw byte = %#x, want %#x", got, want)
	}
}

func requireSymbol(
	t *testing.T,
	encoder *ClientEncoder,
	reader *testBitReader,
	model uint32,
	want byte,
) {
	t.Helper()
	codes, err := buildHuffmanCodes(encoder.tables.data, model)
	if err != nil {
		t.Fatalf("build Huffman model %#x: %v", model, err)
	}

	var prefix uint32
	for length := 1; length <= 31; length++ {
		prefix = prefix<<1 | reader.readBit(t)
		for symbol, code := range codes {
			if int(code&0x1f) == length && code>>(32-length) == prefix {
				if byte(symbol) != want {
					t.Fatalf("model %#x symbol = %#x, want %#x", model, symbol, want)
				}
				return
			}
		}
	}
	t.Fatalf("model %#x has no code matching stream prefix", model)
}
