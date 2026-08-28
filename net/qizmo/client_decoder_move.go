package qizmo

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
)

func (d *ClientDecoder) decodeClientMove(
	reader *bitReader,
	sequence uint32,
	sequenceDelta int,
) (clientMove, error) {
	baseSequence := (sequence - uint32(sequenceDelta)) & sequenceMask
	previous, ok := clientMoveRecordAt(&d.moves, (baseSequence-1)&sequenceMask)
	if !ok {
		return clientMove{}, fmt.Errorf("%w before sequence %d", errUnsupportedClientHistory, baseSequence)
	}
	base, ok := clientMoveRecordAt(&d.moves, baseSequence)
	if !ok {
		return clientMove{}, fmt.Errorf("%w at sequence %d", errUnsupportedClientHistory, baseSequence)
	}

	checksum, err := d.readSymbol(reader, freq.CLCMoveChecksum)
	if err != nil {
		return clientMove{}, fmt.Errorf("decode client move checksum: %w", err)
	}
	lossageDelta, err := d.readSymbol(reader, freq.CLCMoveLossageDelta)
	if err != nil {
		return clientMove{}, fmt.Errorf("decode client move lossage: %w", err)
	}
	move := clientMove{
		checksum: checksum,
		lossage:  base.lossage + lossageDelta,
	}

	firstCommand, commandCount := transmittedClientMoveRange(sequenceDelta)
	for i := 0; i < firstCommand; i++ {
		commandSequence := clientCommandSequence(sequence, i)
		record, ok := clientMoveRecordAt(&d.moves, commandSequence)
		if !ok {
			return clientMove{}, fmt.Errorf("%w at sequence %d", errUnsupportedClientHistory, commandSequence)
		}
		move.commands[i] = record.command
	}

	predictor := newClientMovePredictor(previous.command, base.command, sequenceDelta-commandCount)
	previousMask := uint16(0)
	for i := 0; i < commandCount; i++ {
		command, mask, err := d.decodeClientCommand(reader, &predictor, previousMask)
		if err != nil {
			return clientMove{}, err
		}
		move.commands[firstCommand+i] = command
		previousMask = mask & qizmoprotocol.CMPredictionMask
	}
	return move, nil
}

func (d *ClientDecoder) decodeClientCommand(
	reader *bitReader,
	predictor *clientMovePredictor,
	previousMask uint16,
) (clientCommand, uint16, error) {
	maskLo, err := d.readSymbol(reader, freq.CLCMoveMaskLoXOR)
	if err != nil {
		return clientCommand{}, 0, fmt.Errorf("decode client move mask low byte: %w", err)
	}
	maskHi, err := d.readSymbol(reader, freq.CLCMoveMaskHiXOR)
	if err != nil {
		return clientCommand{}, 0, fmt.Errorf("decode client move mask high byte: %w", err)
	}
	mask := uint16(maskLo) | uint16(maskHi)<<8
	mask ^= previousMask
	if mask&qizmoprotocol.CMInvalid != 0 {
		return clientCommand{}, 0, fmt.Errorf("invalid Qizmo client move mask %#04x", mask)
	}

	var words [len(clientMoveWordModels)]uint16
	for i := range words {
		lowMask, highMask := clientWordMask(i)
		model := clientMoveWordModels[i]
		var low byte
		if mask&lowMask != 0 {
			low, err = d.readSymbol(reader, model.low)
			if err != nil {
				return clientCommand{}, 0, fmt.Errorf("decode client move word %d low byte: %w", i, err)
			}
		}
		words[i] = uint16(int16(int8(low)))
		if mask&highMask != 0 {
			high, err := d.readSymbol(reader, model.high)
			if err != nil {
				return clientCommand{}, 0, fmt.Errorf("decode client move word %d high byte: %w", i, err)
			}
			words[i] += uint16(high) << 8
		}
	}

	var command clientCommand
	for compressedIndex, angleIndex := range qizmoClientAngleOrder {
		delta := words[compressedIndex]
		predicted := predictor.angle[angleIndex] + predictor.angleDelta[angleIndex]
		command.angle[angleIndex] = predicted + delta
		predictor.angleDelta[angleIndex] += delta
		predictor.angle[angleIndex] = command.angle[angleIndex]
	}
	for i := range command.move {
		command.move[i] = predictor.move[i] + words[len(qizmoClientAngleOrder)+i]
		predictor.move[i] = command.move[i]
	}

	command.buttons = predictor.buttons
	if mask&qizmoprotocol.CMButtons != 0 {
		buttonsXOR, err := d.readSymbol(reader, freq.CLCMoveButtonsXOR)
		if err != nil {
			return clientCommand{}, 0, fmt.Errorf("decode client move buttons: %w", err)
		}
		command.buttons ^= buttonsXOR
	}
	predictor.buttons = command.buttons

	command.msec = predictor.msec
	if mask&qizmoprotocol.CMMsec != 0 {
		msecDelta, err := d.readSymbol(reader, freq.CLCMoveMsecDelta)
		if err != nil {
			return clientCommand{}, 0, fmt.Errorf("decode client move msec: %w", err)
		}
		command.msec += msecDelta
	}
	predictor.msec = command.msec

	if mask&qizmoprotocol.CMImpulse != 0 {
		command.impulse, err = d.readSymbol(reader, freq.CLCMoveImpulse)
		if err != nil {
			return clientCommand{}, 0, fmt.Errorf("decode client move impulse: %w", err)
		}
	}
	return command, mask, nil
}

func appendClientMovePayload(packet []byte, move clientMove) []byte {
	packet = append(packet, move.checksum, move.lossage)
	var base clientCommand
	for _, command := range move.commands {
		packet = appendDeltaClientCommand(packet, base, command)
		base = command
	}
	return packet
}

func appendDeltaClientCommand(packet []byte, base, command clientCommand) []byte {
	var commandFlags byte
	if command.angle[pitchAngle] != base.angle[pitchAngle] {
		commandFlags |= protocol.CMAngle1
	}
	if command.angle[yawAngle] != base.angle[yawAngle] {
		commandFlags |= protocol.CMAngle2
	}
	if command.angle[rollAngle] != base.angle[rollAngle] {
		commandFlags |= protocol.CMAngle3
	}
	if command.move[forwardMove] != base.move[forwardMove] {
		commandFlags |= protocol.CMForward
	}
	if command.move[sideMove] != base.move[sideMove] {
		commandFlags |= protocol.CMSide
	}
	if command.move[upMove] != base.move[upMove] {
		commandFlags |= protocol.CMUp
	}
	if command.buttons != base.buttons {
		commandFlags |= protocol.CMButtons
	}
	if command.impulse != base.impulse {
		commandFlags |= protocol.CMImpulse
	}

	packet = append(packet, commandFlags)
	appendWord := func(value uint16) {
		packet = binary.LittleEndian.AppendUint16(packet, value)
	}
	if commandFlags&protocol.CMAngle1 != 0 {
		appendWord(command.angle[pitchAngle])
	}
	if commandFlags&protocol.CMAngle2 != 0 {
		appendWord(command.angle[yawAngle])
	}
	if commandFlags&protocol.CMAngle3 != 0 {
		appendWord(command.angle[rollAngle])
	}
	if commandFlags&protocol.CMForward != 0 {
		appendWord(command.move[forwardMove])
	}
	if commandFlags&protocol.CMSide != 0 {
		appendWord(command.move[sideMove])
	}
	if commandFlags&protocol.CMUp != 0 {
		appendWord(command.move[upMove])
	}
	if commandFlags&protocol.CMButtons != 0 {
		packet = append(packet, command.buttons)
	}
	if commandFlags&protocol.CMImpulse != 0 {
		packet = append(packet, command.impulse)
	}
	return append(packet, command.msec)
}
