package message

import (
	"bytes"
	"testing"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/state"
)

func TestParseOperationFixedPayloads(t *testing.T) {
	opcodes := []byte{
		protocol.SVCNOP,
		protocol.SVCDisconnect,
		protocol.SVCKilledMonster,
		protocol.SVCFoundSecret,
		protocol.SVCSellScreen,
		protocol.SVCSmallKick,
		protocol.SVCBigKick,
		protocol.SVCUpdateStat,
		protocol.SVCUpdateFrags,
		protocol.SVCSpawnStaticSound,
		protocol.SVCIntermission,
		protocol.SVCDamage,
		protocol.SVCMaxSpeed,
		protocol.SVCEntGravity,
		protocol.SVCSetAngle,
		protocol.SVCStopSound,
		protocol.SVCSetPause,
		protocol.SVCCDTrack,
		protocol.SVCChokeCount,
		protocol.SVCUpdateEnterTime,
		protocol.SVCUpdateStatLong,
		protocol.SVCMuzzleFlash,
		protocol.SVCUpdatePing,
		protocol.SVCUpdatePL,
		qizmoprotocol.SVCBlock,
		qizmoprotocol.SVCVoice,
	}

	encoder := &Encoder{}
	for _, opcode := range opcodes {
		size, ok := fixedPayloadSize(opcode)
		if !ok {
			t.Fatalf("opcode %#x has no fixed payload specification", opcode)
		}
		raw := make([]byte, size+1)
		raw[0] = opcode
		for i := 1; i < len(raw); i++ {
			raw[i] = byte(i)
		}

		operation, consumed, ok := encoder.parseOperation(raw)
		if !ok || consumed != len(raw) {
			t.Fatalf("opcode %#x parse = (consumed:%d ok:%v), want (%d, true)",
				opcode, consumed, ok, len(raw))
		}
		if operation.opcode != opcode || !bytes.Equal(operation.data, raw[1:]) {
			t.Fatalf("opcode %#x scan did not preserve payload", opcode)
		}
		if size != 0 {
			if _, _, ok := encoder.parseOperation(raw[:len(raw)-1]); ok {
				t.Fatalf("opcode %#x accepted a truncated payload", opcode)
			}
		}
	}
}

func TestParseOperationVariablePayloads(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{"print", []byte{protocol.SVCPrint, protocol.PrintHigh, 'h', 'i', 0}},
		{"stufftext", []byte{protocol.SVCStuffText, 'x', 0}},
		{"lightstyle", []byte{protocol.SVCLightStyle, 4, 'm', 0}},
		{"centerprint", []byte{protocol.SVCCenterPrint, 'x', 0}},
		{"updateuserinfo", []byte{protocol.SVCUpdateUserInfo, 1, 2, 3, 4, 5, 'u', 0}},
		{"serverinfo", []byte{protocol.SVCServerInfo, 'k', 0, 'v', 0}},
		{"setinfo", []byte{protocol.SVCSetInfo, 3, 'k', 0, 'v', 0}},
		{"qizmo string", []byte{qizmoprotocol.SVCString, 'x', 0}},
		{"nails", []byte{protocol.SVCNails, 1, 1, 2, 3, 4, 5, 6}},
	}

	encoder := &Encoder{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operation, consumed, ok := encoder.parseOperation(test.raw)
			if !ok || consumed != len(test.raw) {
				t.Fatalf("parse = (consumed:%d ok:%v), want (%d, true)",
					consumed, ok, len(test.raw))
			}
			if !bytes.Equal(operation.data, test.raw[1:]) {
				t.Fatalf("payload = %x, want %x", operation.data, test.raw[1:])
			}
		})
	}

	if _, _, ok := encoder.parseOperation([]byte{protocol.SVCPrint, protocol.PrintHigh, 'x'}); ok {
		t.Fatal("unterminated print was accepted")
	}
	if _, _, ok := encoder.parseOperation([]byte{0x55}); ok {
		t.Fatal("unknown opcode was accepted")
	}
}

func TestParseQWDLeadingOperations(t *testing.T) {
	body := []byte{
		protocol.SVCNOP,
		protocol.SVCPrint, protocol.PrintHigh, 'x', 0,
		protocol.SVCPlayerInfo,
	}
	operations, consumed := (&Encoder{}).parseQWDLeadingOperations(body)
	if consumed != len(body)-1 {
		t.Fatalf("consumed = %d, want %d", consumed, len(body)-1)
	}
	if len(operations) != 2 ||
		operations[0].opcode != protocol.SVCNOP ||
		operations[1].opcode != protocol.SVCPrint {
		t.Fatalf("operations = %#v, want nop then print", operations)
	}
}

func TestParseQWDPlayerPrefix(t *testing.T) {
	body := []byte{
		// An additional player before the primary player.
		protocol.SVCPlayerInfo, 1, 1, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		// The primary player.
		protocol.SVCPlayerInfo, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0,
	}
	primary, players, consumed, ok := (&Encoder{
		state:    state.NewPacket(0),
		qwdInput: true,
	}).parseQWDPlayerPrefix(body)
	if !ok || consumed != len(body) {
		t.Fatalf("parse = (consumed:%d ok:%v), want (%d, true)", consumed, ok, len(body))
	}
	if primary.player != 0 || len(players) != 1 || players[0].player != 1 {
		t.Fatalf("primary = %d players = %#v, want primary 0 and additional player 1",
			primary.player, players)
	}
}
