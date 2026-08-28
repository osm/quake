package qizmo

import "github.com/osm/quake/qizmo/state"

func observeClientMovement(packetState *state.Packet, packet []byte) {
	header, ok := readRawClientHeader(packet)
	if !ok {
		return
	}
	sequence := header.sequence & sequenceMask
	operations, _, _ := parseClientOperations(packet[rawClientHeaderSize:])
	for _, operation := range operations {
		if operation.move == nil {
			continue
		}
		for i, command := range operation.move.commands {
			commandSequence := clientCommandSequence(sequence, i)
			packetState.SetCommandScale(commandSequence, command.msec)
		}
	}
}
