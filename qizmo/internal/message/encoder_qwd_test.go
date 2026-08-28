package message

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/state"
)

func TestDeferUpdateUserInfoOperations(t *testing.T) {
	operations := []serviceOperation{
		{opcode: protocol.SVCUpdateUserInfo},
		{opcode: protocol.SVCUpdateFrags},
		{opcode: protocol.SVCUpdateUserInfo},
		{opcode: protocol.SVCUpdatePing},
	}
	got := deferOperations(operations, protocol.SVCUpdateUserInfo)
	want := []byte{
		protocol.SVCUpdateFrags,
		protocol.SVCUpdatePing,
		protocol.SVCUpdateUserInfo,
		protocol.SVCUpdateUserInfo,
	}
	for i, opcode := range want {
		if got[i].opcode != opcode {
			t.Fatalf("operation %d opcode = %#x, want %#x", i, got[i].opcode, opcode)
		}
	}
}

func TestMergeQizmoPrintOperations(t *testing.T) {
	operations := []serviceOperation{
		printOperation(2, "one"),
		printOperation(2, "two"),
		printOperation(0, "low"),
		printOperation(0, "more"),
		printOperation(3, "chat"),
		printOperation(3, "more"),
		printOperation(1, "x"),
	}
	got := mergeQWDPrintOperations(operations)
	want := []serviceOperation{
		printOperation(2, "onetwo"),
		printOperation(0, "low"),
		printOperation(0, "more"),
		printOperation(3, "chat"),
		printOperation(3, "more"),
		printOperation(1, "x"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged operations = %#v, want %#v", got, want)
	}
}

func printOperation(level byte, text string) serviceOperation {
	data := append([]byte{level}, text...)
	return serviceOperation{opcode: protocol.SVCPrint, data: append(data, 0)}
}

func TestSelectPacketBaseUsesPreviousSequence(t *testing.T) {
	st := state.NewPacket(0)
	for _, seq := range []uint32{8, 9} {
		st.SetCommandScale(seq, st.CommandScale(seq))
		st.CurrentPlayers = []state.PlayerRecord{st.DefaultPlayer()}
		st.CommitPlayerSnapshot(seq)
		var entity state.EntityRecord
		state.SetEntityNumber(&entity, uint16(seq))
		st.CommitEntitySnapshot(seq, map[uint16]state.EntityRecord{uint16(seq): entity})
	}

	base := (&Encoder{state: st}).selectPacketBase(10, 8)
	if base.referenceDistance != 2 || base.referenceSequence != 8 {
		t.Fatalf(
			"base = {distance:%d sequence:%d}, want {distance:2 sequence:8}",
			base.referenceDistance,
			base.referenceSequence,
		)
	}
	if len(base.entities) != 1 || state.EntityNumber(base.entities[0]) != 8 {
		t.Fatalf("base entities = %#v, want sequence-8 snapshot", base.entities)
	}
}

func TestSelectPacketBaseRejectsInvalidatedHistory(t *testing.T) {
	st := state.NewPacket(0)
	st.InvalidateHistory(0)
	st.SetCommandScale(8, st.CommandScale(8))
	st.CurrentPlayers = []state.PlayerRecord{st.DefaultPlayer()}
	st.CommitPlayerSnapshot(8)
	st.CommitEntitySnapshot(8, map[uint16]state.EntityRecord{})

	base := (&Encoder{state: st}).selectPacketBase(10, 8)
	if base.referenceDistance != 0 {
		t.Fatalf("base reference distance = %d, want 0", base.referenceDistance)
	}
}

func TestQizmoChatDictionaryPrefixMiss(t *testing.T) {
	dictionary := assets.Embedded().PrintChatStrings
	entries := buildQizmoDictionary(dictionary, 0x140, 0x1fe)
	if entry, ok := qizmoDictionaryLookup(entries, []byte("get quad\n")); ok {
		t.Fatalf("lookup unexpectedly matched symbol %#x %q", entry.symbol, entry.expansion)
	}
}

func TestQizmoStuffTextDictionaryRange(t *testing.T) {
	dictionary := assets.Embedded().StuffTextStrings
	got := (&Encoder{}).planDictionaryString(
		[]byte("topcolor 13\n\x00"),
		freq.SVCStuffText,
		dictionary,
		0x100,
	)
	want := []uint16{'t', 'o', 'p', 0x13d, '1', '3', '\n', 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
}

func TestParseQizmoClearsHighMsecBit(t *testing.T) {
	st := state.NewPacket(0)
	player, consumed, ok := (&Encoder{
		state:    st,
		qwdInput: true,
	}).parseAdditionalPlayer([]byte{
		1, 1, 0, // player and flags
		0, 0, 0, 0, 0, 0, // origin
		0,    // frame
		0xff, // Qizmo shortcut msec
	})
	if !ok || consumed != 11 {
		t.Fatalf("parse = (consumed:%d ok:%v), want (11, true)", consumed, ok)
	}
	if player.record[wire.PlayerMsecOffset] != 0x7f {
		t.Fatalf("msec = %#x, want 0x7f", player.record[wire.PlayerMsecOffset])
	}
	if player.wireMsec != 0xff {
		t.Fatalf("wire msec = %#x, want 0xff", player.wireMsec)
	}
}

func TestQizmoShortcutUsesUnmaskedWireMsec(t *testing.T) {
	st := state.NewPacket(0)
	baseBytes := state.PlayerRecordBytesLE(st.DefaultPlayer())
	baseBytes[wire.PlayerIndexOffset] = 1
	baseBytes[wire.PlayerOriginOffset] = 0x42
	baseBytes[wire.PlayerMsecOffset] = 0x7f
	targetBytes := baseBytes
	targetBytes[wire.PlayerOriginMaskOffset] = 0
	targetBytes[wire.PlayerMsecOffset] = 0x7f
	player := additionalPlayer{
		player:   1,
		wireMsec: 0xff,
		record:   targetBytes,
	}
	base := packetBase{players: []state.PlayerRecord{state.PlayerRecordFromBytesLE(baseBytes)}}

	plan := (&Encoder{state: st, qwdInput: true}).planAdditionalPlayer(
		primaryPlayer{},
		base,
		player,
	)
	if !plan.shortcut {
		t.Fatal("high wire msec with identical visible state did not use shortcut")
	}
	if got := state.PlayerRecordBytesLE(plan.result); got != targetBytes {
		t.Fatalf("shortcut result = %x, want %x", got, targetBytes)
	}

	player.record[wire.PlayerOriginOffset]++
	if plan := (&Encoder{state: st, qwdInput: true}).planAdditionalPlayer(
		primaryPlayer{},
		base,
		player,
	); plan.shortcut {
		t.Fatal("high wire msec with changed visible state used shortcut")
	}
}

func TestQizmoDefersServicesBetweenPlayerRecords(t *testing.T) {
	primary := []byte{
		protocol.SVCPlayerInfo, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0,
	}
	additional := []byte{
		protocol.SVCPlayerInfo, 1, byte(protocol.PFMsec), 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
	entities := []byte{protocol.SVCPacketEntities, 0, 0}
	packet := binary.LittleEndian.AppendUint32(nil, 1)
	packet = binary.LittleEndian.AppendUint32(packet, 1)
	packet = append(packet, primary...)
	packet = append(packet, protocol.SVCNOP)
	packet = append(packet, additional...)
	packet = append(packet, entities...)

	parsed, ok := (&Encoder{
		state:    state.NewPacket(0),
		qwdInput: true,
	}).parsePacket(packet, 0, EncodingOptions{})
	if !ok {
		t.Fatal("packet was not supported")
	}
	want := []byte{protocol.SVCPlayerInfo, protocol.SVCPacketEntities, protocol.SVCNOP}
	if len(parsed.operations) != len(want) {
		t.Fatalf("operation count = %d, want %d", len(parsed.operations), len(want))
	}
	for i, opcode := range want {
		if parsed.operations[i].opcode != opcode {
			t.Fatalf("operation %d opcode = %#x, want %#x", i, parsed.operations[i].opcode, opcode)
		}
	}
}

func TestParseQizmoCanonicalizesPlayerFlags(t *testing.T) {
	wireFlags := uint16(protocol.PFGib)
	data := []byte{
		1,
		byte(wireFlags), byte(wireFlags >> 8),
		0, 0, 0, 0, 0, 0, // origin
		0, // frame
	}
	st := state.NewPacket(0)
	if _, _, ok := (&Encoder{state: st}).parseAdditionalPlayer(data); ok {
		t.Fatal("link encoder accepted playerinfo without PF_MSEC")
	}
	player, consumed, ok := (&Encoder{state: st, qwdInput: true}).parseAdditionalPlayer(data)
	if !ok || consumed != len(data) {
		t.Fatalf("exact parse = (consumed:%d ok:%v), want (%d, true)", consumed, ok, len(data))
	}
	if player.flags != protocol.PFMsec || player.record[wire.PlayerMsecOffset] != 0 {
		t.Fatalf("canonical flags = %#x msec = %#x, want PF_MSEC and zero", player.flags, player.record[wire.PlayerMsecOffset])
	}
}

func TestBuildDecodedPlayerInfoFlagsOmitsButtons(t *testing.T) {
	var record state.PlayerRecordBytes
	record[wire.PlayerButtonsOffset] = 0x77
	flags, extra := buildDecodedPlayerInfoFlags(&record, 0)
	if flags != protocol.PFMsec || extra != 0 {
		t.Fatalf("button-only flags = %#x extra = %#x, want PF_MSEC and zero", flags, extra)
	}

	record[wire.PlayerAngleOffset+2] = 1
	flags, extra = buildDecodedPlayerInfoFlags(&record, 0)
	if flags != protocol.PFMsec|protocol.PFCommand || extra != protocol.CMAngle1 {
		t.Fatalf("angle/button flags = %#x extra = %#x, want PF_MSEC|PF_COMMAND and CM_ANGLE1", flags, extra)
	}
}

func TestParseQizmoDiscardsUntrackedAngle3(t *testing.T) {
	data := []byte{
		1, 3, 0, // player and PF_MSEC | PF_COMMAND
		0x19, 0x18, 0xe6, 0xf9, 0x40, 0x03, // origin
		0x0e, // frame
		0x23, // packet msec
		protocol.CMAngle1 | protocol.CMAngle2 | protocol.CMAngle3,
		0x8b, 0x06, // angle 1
		0x79, 0x39, // angle 2
		0x33, 0x33, // angle 3, discarded by Qizmo
		0x0d, // command msec
	}
	st := state.NewPacket(0)
	if _, _, ok := (&Encoder{state: st}).parseAdditionalPlayer(data); ok {
		t.Fatal("link encoder accepted an unrepresentable Angle 3")
	}
	player, consumed, ok := (&Encoder{state: st, qwdInput: true}).parseAdditionalPlayer(data)
	if !ok || consumed != len(data) {
		t.Fatalf("exact parse = (consumed:%d ok:%v), want (%d, true)", consumed, ok, len(data))
	}
	if want := byte(protocol.CMAngle1 | protocol.CMAngle2); player.commandFlags != want {
		t.Fatalf("canonical command flags = %#x, want %#x", player.commandFlags, want)
	}
	if got := state.PlayerRecordUint16(&player.record, wire.PlayerAngleOffset+2); got != 0x068b {
		t.Fatalf("angle 1 = %#x, want 0x068b", got)
	}
	if got := state.PlayerRecordUint16(&player.record, wire.PlayerAngleOffset); got != 0x3979 {
		t.Fatalf("angle 2 = %#x, want 0x3979", got)
	}
}
