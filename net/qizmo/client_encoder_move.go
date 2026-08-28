package qizmo

import (
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/packed"
)

func (e *ClientEncoder) encodeClientMove(
	writer *bitWriter,
	move clientMove,
	sequence uint32,
	sequenceDelta int,
) error {
	baseSequence := (sequence - uint32(sequenceDelta)) & sequenceMask
	previous, ok := clientMoveRecordAt(&e.moves, (baseSequence-1)&sequenceMask)
	if !ok {
		return errUnsupportedClientHistory
	}
	base, ok := clientMoveRecordAt(&e.moves, baseSequence)
	if !ok {
		return errUnsupportedClientHistory
	}

	if err := e.writeSymbol(writer, freq.CLCMoveChecksum, move.checksum); err != nil {
		return err
	}
	if err := e.writeSymbol(writer, freq.CLCMoveLossageDelta, move.lossage-base.lossage); err != nil {
		return err
	}

	firstCommand, commandCount := transmittedClientMoveRange(sequenceDelta)
	predictor := newClientMovePredictor(previous.command, base.command, sequenceDelta-commandCount)
	previousMask := uint16(0)
	for i := 0; i < commandCount; i++ {
		delta := predictor.plan(move.commands[firstCommand+i])
		if err := e.encodeClientCommandDelta(writer, delta, previousMask); err != nil {
			return err
		}
		previousMask = delta.mask & qizmoprotocol.CMPredictionMask
	}
	return nil
}

type clientCommandDelta struct {
	mask    uint16
	angle   [3]uint16
	move    [3]uint16
	buttons byte
	impulse byte
	msec    byte
}

func (p *clientMovePredictor) plan(command clientCommand) clientCommandDelta {
	var delta clientCommandDelta
	for compressedIndex, angleIndex := range qizmoClientAngleOrder {
		predicted := p.angle[angleIndex] + p.angleDelta[angleIndex]
		delta.angle[angleIndex] = command.angle[angleIndex] - predicted
		setClientWordMask(&delta.mask, delta.angle[angleIndex], compressedIndex)
		p.angleDelta[angleIndex] += delta.angle[angleIndex]
		p.angle[angleIndex] = command.angle[angleIndex]
	}
	for i := range p.move {
		delta.move[i] = command.move[i] - p.move[i]
		setClientWordMask(&delta.mask, delta.move[i], len(qizmoClientAngleOrder)+i)
		p.move[i] = command.move[i]
	}
	delta.buttons = command.buttons ^ p.buttons
	if delta.buttons != 0 {
		delta.mask |= qizmoprotocol.CMButtons
	}
	p.buttons = command.buttons

	delta.msec = command.msec - p.msec
	if delta.msec != 0 {
		delta.mask |= qizmoprotocol.CMMsec
	}
	p.msec = command.msec

	delta.impulse = command.impulse
	if delta.impulse != 0 {
		delta.mask |= qizmoprotocol.CMImpulse
	}
	return delta
}

func setClientWordMask(mask *uint16, delta uint16, index int) {
	lowMask, highMask := clientWordMask(index)
	low := byte(delta)
	if low != 0 {
		*mask |= lowMask
	}
	if int16(delta) != int16(int8(low)) {
		*mask |= highMask
	}
}

func (e *ClientEncoder) encodeClientCommandDelta(
	writer *bitWriter,
	delta clientCommandDelta,
	previousMask uint16,
) error {
	maskXOR := delta.mask ^ previousMask
	if err := e.writeSymbol(writer, freq.CLCMoveMaskLoXOR, byte(maskXOR)); err != nil {
		return err
	}
	if err := e.writeSymbol(writer, freq.CLCMoveMaskHiXOR, byte(maskXOR>>8)); err != nil {
		return err
	}

	words := [...]uint16{
		delta.angle[yawAngle],
		delta.angle[pitchAngle],
		delta.angle[rollAngle],
		delta.move[forwardMove], delta.move[sideMove], delta.move[upMove],
	}
	for i, value := range words {
		lowMask, highMask := clientWordMask(i)
		model := clientMoveWordModels[i]
		low, high := packed.SplitDelta16(value)
		if delta.mask&lowMask != 0 {
			if err := e.writeSymbol(writer, model.low, low); err != nil {
				return err
			}
		}
		if delta.mask&highMask != 0 {
			if err := e.writeSymbol(writer, model.high, high); err != nil {
				return err
			}
		}
	}
	if delta.mask&qizmoprotocol.CMButtons != 0 {
		if err := e.writeSymbol(writer, freq.CLCMoveButtonsXOR, delta.buttons); err != nil {
			return err
		}
	}
	if delta.mask&qizmoprotocol.CMMsec != 0 {
		if err := e.writeSymbol(writer, freq.CLCMoveMsecDelta, delta.msec); err != nil {
			return err
		}
	}
	if delta.mask&qizmoprotocol.CMImpulse != 0 {
		if err := e.writeSymbol(writer, freq.CLCMoveImpulse, delta.impulse); err != nil {
			return err
		}
	}
	return nil
}
