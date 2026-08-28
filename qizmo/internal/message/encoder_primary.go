package message

import (
	"encoding/binary"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/state"
)

const primaryPlayerFlagMask = uint16(
	protocol.PFVelocity1 |
		protocol.PFVelocity2 |
		protocol.PFVelocity3 |
		protocol.PFModel |
		protocol.PFSkinNum |
		protocol.PFEffects |
		protocol.PFWeaponFrame |
		protocol.PFDead,
)

const (
	primaryPlayerFlagsOffset  = 2
	primaryPlayerOriginOffset = 4
	primaryPlayerFrameOffset  = 10
	primaryPlayerFixedSize    = primaryPlayerFrameOffset + 1
)

type primaryPlayer struct {
	player      byte
	flags       uint16
	origin      [3]int16
	frame       byte
	velocity    [3]int16
	model       byte
	skinNum     byte
	effects     byte
	weaponFrame byte
}

func (e *Encoder) parsePrimaryPlayer(body []byte) (primaryPlayer, int, bool) {
	player := primaryPlayer{model: e.state.PlayerModelIndex}
	if len(body) < primaryPlayerFixedSize ||
		body[0] != protocol.SVCPlayerInfo ||
		body[1] != e.state.PlayerIndex {
		return player, 0, false
	}
	player.player = body[1]
	player.flags = binary.LittleEndian.Uint16(body[primaryPlayerFlagsOffset:primaryPlayerOriginOffset])
	if player.flags&^primaryPlayerFlagMask != 0 {
		return player, 0, false
	}
	for axis := range player.origin {
		offset := primaryPlayerOriginOffset + axis*2
		player.origin[axis] = int16(binary.LittleEndian.Uint16(body[offset:]))
	}
	player.frame = body[primaryPlayerFrameOffset]
	off := primaryPlayerFixedSize

	for i, field := range wire.PlayerVelocityFields {
		if player.flags&field.Mask == 0 {
			continue
		}
		if off+field.Size > len(body) {
			return primaryPlayer{}, 0, false
		}
		player.velocity[i] = int16(binary.LittleEndian.Uint16(body[off : off+2]))
		off += field.Size
	}
	byteFields := [4]*byte{&player.model, &player.skinNum, &player.effects, &player.weaponFrame}
	for i, field := range wire.PlayerByteFields {
		if player.flags&field.Mask == 0 {
			continue
		}
		if off >= len(body) {
			return primaryPlayer{}, 0, false
		}
		*byteFields[i] = body[off]
		off++
	}

	expectedFlags := player.flags & protocol.PFDead
	for i, field := range wire.PlayerVelocityFields {
		if player.velocity[i] != 0 {
			expectedFlags |= field.Mask
		}
	}
	if player.model != e.state.PlayerModelIndex {
		expectedFlags |= protocol.PFModel
	}
	if player.skinNum != 0 {
		expectedFlags |= protocol.PFSkinNum
	}
	if player.effects != 0 {
		expectedFlags |= protocol.PFEffects
	}
	if player.weaponFrame != 0 {
		expectedFlags |= protocol.PFWeaponFrame
	}
	if expectedFlags != player.flags {
		return primaryPlayer{}, 0, false
	}
	if player.flags&protocol.PFModel != 0 {
		if _, ok := e.state.ModelRemapIndex(player.model); !ok {
			return primaryPlayer{}, 0, false
		}
	}
	return player, off, true
}

func (e *Encoder) encodeSVCPlayerInfo(
	enc *rangeenc.Encoder,
	plan primaryPlayerPlan,
) error {
	player := plan.player

	if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoBackReference, plan.referenceDistance); err != nil {
		return err
	}
	maskDeltas := [...]byte{
		plan.originMask ^ plan.base[wire.PlayerOriginMaskOffset],
		plan.stateMask ^ plan.base[wire.PlayerStateMaskOffset],
		plan.motionMask ^ plan.base[wire.PlayerMotionMaskOffset],
	}
	for i, row := range primaryPlayerMaskXORRows {
		if err := enc.EncodeFreqByte(e.ft, row, maskDeltas[i]); err != nil {
			return err
		}
	}

	for i, rows := range playerOriginDeltaRows {
		if err := e.encodeWordDelta(enc, plan.originDelta[i], rows.low, rows.high); err != nil {
			return err
		}
	}
	if plan.originMask&wire.PlayerFrameDelta != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoFrameDelta, plan.frameDelta); err != nil {
			return err
		}
	}

	if plan.stateMask&wire.PlayerModelRemap != 0 {
		remapIndex, _ := e.state.ModelRemapIndex(player.model)
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoModelRemapIndex, remapIndex); err != nil {
			return err
		}
	}
	if plan.stateMask&wire.PlayerSkinNumSet != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoSkinNumSet, player.skinNum); err != nil {
			return err
		}
	}
	if plan.stateMask&wire.PlayerEffectsXOR != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoEffectsXOR, plan.effectsXOR); err != nil {
			return err
		}
	}

	for i, rows := range playerVelocityDeltaRows {
		if err := e.encodeWordDelta(enc, plan.accumulatorDelta[i], rows.low, rows.high); err != nil {
			return err
		}
	}
	if plan.motionMask&wire.PlayerWeaponFrameDelta != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoWeaponFrameDelta, plan.weaponFrameDelta); err != nil {
			return err
		}
	}
	return nil
}

type primaryPlayerPlan struct {
	player            primaryPlayer
	base              state.PlayerRecordBytes
	referenceDistance byte
	originMask        byte
	stateMask         byte
	motionMask        byte
	originDelta       [3]packed.WordDelta
	accumulatorDelta  [3]packed.WordDelta
	frameDelta        byte
	effectsXOR        byte
	weaponFrameDelta  byte
	result            state.PlayerRecord
}

func (e *Encoder) planPrimaryPlayer(
	player primaryPlayer,
	base packetBase,
) primaryPlayerPlan {
	baseBytes := state.PlayerRecordBytesLE(base.primary)
	predictedOrigin := predictedPlayerOrigin(&baseBytes, base.predictionScale)
	velocityAccumulators, accumulatorDeltas := playerVelocityAccumulatorDeltas(&baseBytes, player.velocity)
	plan := primaryPlayerPlan{
		player:            player,
		base:              baseBytes,
		referenceDistance: base.referenceDistance,
		accumulatorDelta:  accumulatorDeltas,
		originDelta: [3]packed.WordDelta{
			packed.SplitWordDelta(player.origin[0], predictedOrigin[0]),
			packed.SplitWordDelta(player.origin[1], predictedOrigin[1]),
			packed.SplitWordDelta(player.origin[2], predictedOrigin[2]),
		},
		frameDelta:       player.frame - baseBytes[wire.PlayerFrameOffset],
		effectsXOR:       player.effects ^ baseBytes[wire.PlayerEffectsOffset],
		weaponFrameDelta: player.weaponFrame - baseBytes[wire.PlayerWeaponFrameOffset],
	}

	for axis, delta := range plan.originDelta {
		setWordDeltaMask(&plan.originMask, delta, axis)
	}
	if plan.frameDelta != 0 {
		plan.originMask |= wire.PlayerFrameDelta
	}
	if player.model != baseBytes[wire.PlayerModelOffset] {
		plan.stateMask |= wire.PlayerModelRemap
	}
	if player.skinNum != baseBytes[wire.PlayerSkinNumOffset] {
		plan.stateMask |= wire.PlayerSkinNumSet
	}
	if plan.effectsXOR != 0 {
		plan.stateMask |= wire.PlayerEffectsXOR
	}
	for axis, delta := range plan.accumulatorDelta {
		setWordDeltaMask(&plan.motionMask, delta, axis)
	}
	if plan.weaponFrameDelta != 0 {
		plan.motionMask |= wire.PlayerWeaponFrameDelta
	}
	if player.flags&protocol.PFDead != 0 {
		plan.motionMask |= wire.PlayerDead
	}

	resultBytes := baseBytes
	resultBytes[wire.PlayerOriginMaskOffset] = plan.originMask & wire.PlayerOriginHistoryMask
	resultBytes[wire.PlayerStateMaskOffset] = plan.stateMask & wire.PlayerStateHistoryMask
	resultBytes[wire.PlayerMotionMaskOffset] = plan.motionMask & wire.PlayerMotionHistoryMask
	for axis, value := range player.origin {
		state.SetPlayerRecordUint16(&resultBytes, wire.PlayerOriginOffset+axis*2, uint16(value))
	}
	resultBytes[wire.PlayerFrameOffset] = player.frame
	resultBytes[wire.PlayerIndexOffset] = player.player
	resultBytes[wire.PlayerModelOffset] = player.model
	resultBytes[wire.PlayerSkinNumOffset] = player.skinNum
	resultBytes[wire.PlayerEffectsOffset] = player.effects
	for axis, field := range wire.PlayerVelocityFields {
		state.SetPlayerRecordUint16(&resultBytes, field.RecordOffset, uint16(player.velocity[axis]))
		accumulatorOffset := wire.PlayerVelocityAccumulatorOffset + axis*2
		state.SetPlayerRecordUint16(&resultBytes, accumulatorOffset, uint16(velocityAccumulators[axis]))
	}
	resultBytes[wire.PlayerWeaponFrameOffset] = player.weaponFrame
	plan.result = state.PlayerRecordFromBytesLE(resultBytes)
	return plan
}

func setWordDeltaMask(mask *byte, delta packed.WordDelta, index int) {
	low := byte(1 << uint(index*2))
	if delta.Low != 0 {
		*mask |= low
	}
	if delta.High != 0 {
		*mask |= low << 1
	}
}

func (e *Encoder) encodeWordDelta(
	enc *rangeenc.Encoder,
	delta packed.WordDelta,
	loRow, hiRow uint32,
) error {
	if delta.Low != 0 {
		if err := enc.EncodeFreqByte(e.ft, loRow, delta.Low); err != nil {
			return err
		}
	}
	if delta.High != 0 {
		if err := enc.EncodeFreqByte(e.ft, hiRow, delta.High); err != nil {
			return err
		}
	}
	return nil
}
