package qizmo

import (
	"encoding/binary"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
)

const (
	clientMoveCommandCount = 3
	clientMoveHistorySize  = 32
	clientMoveHistoryMask  = clientMoveHistorySize - 1
)

const (
	clientMoveChecksumOffset = iota
	clientMoveLossageOffset
	clientMoveHeaderSize
)

const (
	pitchAngle = iota
	yawAngle
	rollAngle
)

const (
	forwardMove = iota
	sideMove
	upMove
)

type clientCommand struct {
	angle   [3]uint16
	move    [3]uint16
	buttons byte
	impulse byte
	msec    byte
}

type clientMove struct {
	checksum byte
	lossage  byte
	commands [clientMoveCommandCount]clientCommand
}

type clientMoveRecord struct {
	sequence uint32
	command  clientCommand
	lossage  byte
	valid    bool
}

type clientMoveHistory [clientMoveHistorySize]clientMoveRecord

func clientMoveRecordAt(
	history *clientMoveHistory,
	sequence uint32,
) (clientMoveRecord, bool) {
	record := history[sequence&clientMoveHistoryMask]
	return record, record.valid && record.sequence == sequence
}

func parseClientMove(data []byte) (clientMove, int, bool) {
	var move clientMove
	if len(data) < clientMoveHeaderSize {
		return move, 0, false
	}
	move.checksum = data[clientMoveChecksumOffset]
	move.lossage = data[clientMoveLossageOffset]
	offset := clientMoveHeaderSize
	var base clientCommand
	for i := range move.commands {
		command, consumed, ok := parseDeltaClientCommand(data[offset:], base)
		if !ok {
			return clientMove{}, 0, false
		}
		move.commands[i] = command
		base = command
		offset += consumed
	}
	return move, offset, true
}

func parseDeltaClientCommand(data []byte, base clientCommand) (clientCommand, int, bool) {
	if len(data) < 1 {
		return clientCommand{}, 0, false
	}
	command := base
	commandFlags := data[0]
	offset := 1
	readWord := func(target *uint16) bool {
		if len(data)-offset < 2 {
			return false
		}
		*target = binary.LittleEndian.Uint16(data[offset:])
		offset += 2
		return true
	}
	readByte := func(target *byte) bool {
		if len(data)-offset < 1 {
			return false
		}
		*target = data[offset]
		offset++
		return true
	}

	if commandFlags&protocol.CMAngle1 != 0 && !readWord(&command.angle[pitchAngle]) {
		return clientCommand{}, 0, false
	}
	if commandFlags&protocol.CMAngle2 != 0 && !readWord(&command.angle[yawAngle]) {
		return clientCommand{}, 0, false
	}
	if commandFlags&protocol.CMAngle3 != 0 && !readWord(&command.angle[rollAngle]) {
		return clientCommand{}, 0, false
	}
	if commandFlags&protocol.CMForward != 0 && !readWord(&command.move[forwardMove]) {
		return clientCommand{}, 0, false
	}
	if commandFlags&protocol.CMSide != 0 && !readWord(&command.move[sideMove]) {
		return clientCommand{}, 0, false
	}
	if commandFlags&protocol.CMUp != 0 && !readWord(&command.move[upMove]) {
		return clientCommand{}, 0, false
	}
	if commandFlags&protocol.CMButtons != 0 && !readByte(&command.buttons) {
		return clientCommand{}, 0, false
	}
	if commandFlags&protocol.CMImpulse != 0 && !readByte(&command.impulse) {
		return clientCommand{}, 0, false
	}
	// Protocol 27 and later always append msec. Qizmo's link codec only
	// supports this QuakeWorld user-command layout.
	if !readByte(&command.msec) {
		return clientCommand{}, 0, false
	}
	return command, offset, true
}

func transmittedClientMoveRange(sequenceDelta int) (first, count int) {
	count = min(sequenceDelta, clientMoveCommandCount)
	return clientMoveCommandCount - count, count
}

type clientMovePredictor struct {
	angle      [3]uint16
	angleDelta [3]uint16
	move       [3]uint16
	buttons    byte
	msec       byte
}

// Qizmo's private move-mask order differs from delta_usercmd's wire order:
// yaw is encoded before pitch, followed by roll.
var qizmoClientAngleOrder = [...]int{yawAngle, pitchAngle, rollAngle}

type clientMoveWordModel struct {
	low  uint32
	high uint32
}

var clientMoveWordModels = [...]clientMoveWordModel{
	{freq.CLCMoveYawDeltaLo, freq.CLCMoveYawDeltaHi},
	{freq.CLCMovePitchDeltaLo, freq.CLCMovePitchDeltaHi},
	{freq.CLCMoveRollDeltaLo, freq.CLCMoveRollDeltaHi},
	{freq.CLCMoveForwardDeltaLo, freq.CLCMoveForwardDeltaHi},
	{freq.CLCMoveSideDeltaLo, freq.CLCMoveSideDeltaHi},
	{freq.CLCMoveUpDeltaLo, freq.CLCMoveUpDeltaHi},
}

func newClientMovePredictor(previous, base clientCommand, skipped int) clientMovePredictor {
	predictor := clientMovePredictor{
		move:    base.move,
		buttons: base.buttons,
		msec:    base.msec,
	}
	for i := range predictor.angle {
		predictor.angleDelta[i] = base.angle[i] - previous.angle[i]
		predictor.angle[i] = base.angle[i] + predictor.angleDelta[i]*uint16(skipped)
	}
	return predictor
}

func clientWordMask(index int) (uint16, uint16) {
	low := uint16(1 << uint(index*2))
	return low, low << 1
}
