package svc

import (
	"bytes"
	"testing"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/nops"
	"github.com/osm/quake/packet/command/qizmoblock"
	"github.com/osm/quake/packet/command/qizmostring"
	"github.com/osm/quake/packet/command/qizmovoice"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
	"github.com/osm/quake/protocol/qizmo"
)

func TestParseGameDataQizmoServices(t *testing.T) {
	stringCommand := &qizmostring.Command{String: "qizmo"}
	blockCommand := &qizmoblock.Command{Data: make([]byte, qizmo.SVCBlockPayloadSize)}
	voiceCommand := &qizmovoice.Command{Data: make([]byte, qizmo.SVCVoicePayloadSize)}
	for i := range blockCommand.Data {
		blockCommand.Data[i] = byte(i)
	}
	for i := range voiceCommand.Data {
		voiceCommand.Data[i] = byte(i + 1)
	}

	packet := make([]byte, protocol.QWServerPacketHeaderSize)
	wantRaw := [][]byte{
		stringCommand.Bytes(),
		blockCommand.Bytes(),
		voiceCommand.Bytes(),
		{protocol.SVCNOP},
	}
	for _, raw := range wantRaw {
		packet = append(packet, raw...)
	}

	got, err := ParseGameDataWithOptions(
		context.New(),
		packet,
		Options{QizmoCompatibility: true},
	)
	if err != nil {
		t.Fatalf("parse Qizmo services: %v", err)
	}
	if len(got.Commands) != len(wantRaw) {
		t.Fatalf("command count = %d, want %d", len(got.Commands), len(wantRaw))
	}
	if _, ok := got.Commands[0].(*qizmostring.Command); !ok {
		t.Fatalf("command 0 has type %T", got.Commands[0])
	}
	if _, ok := got.Commands[1].(*qizmoblock.Command); !ok {
		t.Fatalf("command 1 has type %T", got.Commands[1])
	}
	if _, ok := got.Commands[2].(*qizmovoice.Command); !ok {
		t.Fatalf("command 2 has type %T", got.Commands[2])
	}
	if _, ok := got.Commands[3].(*nops.Command); !ok {
		t.Fatalf("command 3 has type %T", got.Commands[3])
	}
	for i := range wantRaw {
		if !bytes.Equal(got.RawCmds[i], wantRaw[i]) {
			t.Fatalf("raw command %d mismatch", i)
		}
	}
}

func TestParseGameDataQizmoStopsBeforeOtherExtensions(t *testing.T) {
	packet := make([]byte, protocol.QWServerPacketHeaderSize)
	packet = append(packet, protocol.SVCNOP, fte.SVCVoiceChat, protocol.SVCNOP)

	got, err := ParseGameDataWithOptions(
		context.New(),
		packet,
		Options{QizmoCompatibility: true},
	)
	if err != nil {
		t.Fatalf("parse compatibility packet: %v", err)
	}
	if len(got.Commands) != 1 {
		t.Fatalf("command count = %d, want 1", len(got.Commands))
	}
	if _, ok := got.Commands[0].(*nops.Command); !ok {
		t.Fatalf("command 0 has type %T", got.Commands[0])
	}
}
