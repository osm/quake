package qwz

import (
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/playerinfo"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
)

// canonicalizeConnectionPacket mirrors the connection-state changes made by
// Qizmo's QWD loader before compression.
func canonicalizeConnectionPacket(
	ctx *context.Context,
	packet []byte,
	disconnected bool,
) ([]byte, bool) {
	if len(packet) < protocol.QWServerPacketHeaderSize {
		return packet, false
	}
	gameData, err := svc.ParseGameDataWithOptions(
		ctx,
		packet,
		svc.Options{QizmoCompatibility: true},
	)
	if err != nil {
		return packet, false
	}

	if !disconnected {
		for _, raw := range gameData.RawCmds {
			if len(raw) != 0 && raw[0] == protocol.SVCDisconnect {
				return replaceDisconnectWithNOP(packet, gameData.RawCmds), true
			}
		}
		return packet, false
	}

	canonical := append([]byte(nil), packet[:protocol.QWServerPacketHeaderSize]...)
	for i, raw := range gameData.RawCmds {
		if _, ok := gameData.Commands[i].(*playerinfo.Command); !ok {
			canonical = append(canonical, raw...)
		}
	}
	return canonical, false
}

func replaceDisconnectWithNOP(packet []byte, commands [][]byte) []byte {
	canonical := append([]byte(nil), packet[:protocol.QWServerPacketHeaderSize]...)
	for _, raw := range commands {
		if len(raw) != 0 && raw[0] == protocol.SVCDisconnect {
			canonical = append(canonical, protocol.SVCNOP)
		} else {
			canonical = append(canonical, raw...)
		}
	}
	return canonical
}
