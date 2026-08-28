package message

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/state"
)

const (
	additionalPlayerFlagMask = uint16(
		protocol.PFMsec |
			protocol.PFCommand |
			protocol.PFVelocity1 |
			protocol.PFVelocity2 |
			protocol.PFVelocity3 |
			protocol.PFModel |
			protocol.PFSkinNum |
			protocol.PFEffects |
			protocol.PFWeaponFrame |
			protocol.PFDead,
	)
	supportedPlayerCommandFlags = byte(
		protocol.CMAngle1 |
			protocol.CMAngle2 |
			protocol.CMForward |
			protocol.CMSide |
			protocol.CMUp |
			protocol.CMImpulse,
	)
)

const (
	additionalPlayerFlagsOffset  = 1
	additionalPlayerOriginOffset = 3
	additionalPlayerFrameOffset  = 9
	additionalPlayerFixedSize    = additionalPlayerFrameOffset + 1
)

// additionalPlayer is the canonical svc_playerinfo state emitted by
// Qizmo after the packet's primary player. It is encoded against the selected
// packet-history snapshot when one is available.
type additionalPlayer struct {
	player       byte
	flags        uint16
	commandFlags byte
	wireMsec     byte
	record       state.PlayerRecordBytes
}

func (e *Encoder) parseAdditionalPlayer(data []byte) (additionalPlayer, int, bool) {
	var player additionalPlayer
	if len(data) < additionalPlayerFixedSize {
		return player, 0, false
	}
	player.player = data[0]
	wireFlags := binary.LittleEndian.Uint16(data[additionalPlayerFlagsOffset:additionalPlayerOriginOffset])
	player.flags = wireFlags
	if !e.qwdInput && (wireFlags&^additionalPlayerFlagMask != 0 || wireFlags&protocol.PFMsec == 0) {
		return additionalPlayer{}, 0, false
	}
	player.record = state.PlayerRecordBytesLE(e.state.DefaultPlayer())
	player.record[wire.PlayerIndexOffset] = player.player
	copy(
		player.record[wire.PlayerOriginOffset:wire.PlayerFrameOffset],
		data[additionalPlayerOriginOffset:additionalPlayerFrameOffset],
	)
	player.record[wire.PlayerFrameOffset] = data[additionalPlayerFrameOffset]
	if player.flags&protocol.PFDead != 0 {
		player.record[wire.PlayerMotionMaskOffset] = wire.PlayerDead
	}
	off := additionalPlayerFixedSize
	wireMsec := byte(0)
	hasWireMsec := wireFlags&protocol.PFMsec != 0
	if hasWireMsec {
		if off >= len(data) {
			return additionalPlayer{}, 0, false
		}
		wireMsec = data[off]
		off++
	}
	player.wireMsec = wireMsec
	player.record[wire.PlayerMsecOffset] = wireMsec

	if wireFlags&protocol.PFCommand != 0 {
		var ok bool
		off, ok = e.parseAdditionalPlayerCommand(data, off, &player)
		if !ok {
			return additionalPlayer{}, 0, false
		}
	}

	var ok bool
	off, ok = parseAdditionalPlayerFields(data, off, &player)
	if !ok {
		return additionalPlayer{}, 0, false
	}

	flags, commandFlags := buildPlayerInfoFlags(&player.record, e.state.PlayerModelIndex)
	if e.qwdInput {
		// Qizmo accepts non-canonical playerinfo flags and compresses the
		// reconstructed state. That adds a missing PF_MSEC and drops control
		// flags which carry no playerinfo payload.
		player.flags = flags
		player.commandFlags = commandFlags
	} else if flags != player.flags || commandFlags != player.commandFlags {
		return additionalPlayer{}, 0, false
	}
	if hasWireMsec && wireMsec&wire.PlayerMsecShortcut != 0 {
		if !e.qwdInput {
			return additionalPlayer{}, 0, false
		}
		player.record[wire.PlayerMsecOffset] &^= wire.PlayerMsecShortcut
	}
	if !e.qwdInput {
		wireRecord := player.record
		wireRecord[wire.PlayerMsecOffset] = wireMsec
		canonical := appendPlayerInfoRecord(nil, player.player, flags, commandFlags, &wireRecord)
		if !bytes.Equal(data[:off], canonical) {
			return additionalPlayer{}, 0, false
		}
	}
	if player.flags&protocol.PFModel != 0 {
		if _, ok := e.state.ModelRemapIndex(player.record[wire.PlayerModelOffset]); !ok {
			return additionalPlayer{}, 0, false
		}
	}
	return player, off, true
}

func (e *Encoder) parseAdditionalPlayerCommand(
	data []byte,
	offset int,
	player *additionalPlayer,
) (int, bool) {
	if offset >= len(data) {
		return 0, false
	}
	commandFlags := data[offset]
	player.commandFlags = commandFlags
	offset++

	supported := supportedPlayerCommandFlags
	// Qizmo consumes Angle 3 but has no corresponding history field, and it
	// tracks Buttons without reproducing them in the decoded packet. QWD input
	// mirrors both quirks; the link encoder retains such packets raw.
	if e.qwdInput {
		supported |= protocol.CMAngle3 | protocol.CMButtons
	}
	if commandFlags&^supported != 0 {
		return 0, false
	}
	for _, field := range wire.PlayerCommandFields {
		if commandFlags&field.Mask == 0 {
			continue
		}
		if offset+field.Size > len(data) {
			return 0, false
		}
		if field.RecordOffset != wire.UntrackedRecordOffset {
			recordOffset := field.RecordOffset
			copy(player.record[recordOffset:recordOffset+field.Size], data[offset:offset+field.Size])
		}
		offset += field.Size
	}
	if offset >= len(data) {
		return 0, false
	}
	player.record[wire.PlayerCommandMsecOffset] = data[offset]
	offset++
	player.commandFlags &^= protocol.CMAngle3
	return offset, true
}

func parseAdditionalPlayerFields(
	data []byte,
	offset int,
	player *additionalPlayer,
) (int, bool) {
	for _, field := range wire.PlayerVelocityFields {
		if player.flags&field.Mask == 0 {
			continue
		}
		if offset+field.Size > len(data) {
			return 0, false
		}
		recordOffset := field.RecordOffset
		copy(player.record[recordOffset:recordOffset+field.Size], data[offset:offset+field.Size])
		offset += field.Size
	}
	for _, field := range wire.PlayerByteFields {
		if player.flags&field.Mask == 0 {
			continue
		}
		if offset >= len(data) {
			return 0, false
		}
		player.record[field.RecordOffset] = data[offset]
		offset++
	}
	return offset, true
}

func (e *Encoder) encodeSVCPlayerInfoDeltas(
	enc *rangeenc.Encoder,
	ctx *encodingContext,
	plans []additionalPlayerPlan,
) error {
	lastPlayer := noPlayerIndex
	for i, plan := range plans {
		player := plan.player
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoNumDelta, player.player-lastPlayer); err != nil {
			return fmt.Errorf("player %d number: %w", i, err)
		}
		lastPlayer = player.player
		if err := e.encodeAdditionalPlayer(enc, plan); err != nil {
			return fmt.Errorf("player %d slot %d: %w", i, player.player, err)
		}
		ctx.currentPlayers = append(ctx.currentPlayers, plan.result)
		if plan.shortcut && i+1 < len(plans) {
			// The original compressor closes the current player-delta list after
			// a high-bit msec shortcut, then opens another svc_playerinfo list.
			// The running player number deliberately survives that boundary.
			if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoNumDelta, 0); err != nil {
				return fmt.Errorf("player %d shortcut terminator: %w", i, err)
			}
			if err := enc.EncodeFreqByte(e.ft, freq.SVCType, protocol.SVCPlayerInfo); err != nil {
				return fmt.Errorf("player %d shortcut continuation: %w", i, err)
			}
		}
	}
	if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoNumDelta, 0); err != nil {
		return fmt.Errorf("player terminator: %w", err)
	}
	return nil
}

func (e *Encoder) encodeSVCUpdatePing(
	enc *rangeenc.Encoder,
	ctx *encodingContext,
	data []byte,
) error {
	delta := data[0] - ctx.lastPingPlayerIndex
	ctx.lastPingPlayerIndex = data[0]
	return e.encodeRows(enc, []byte{delta, data[1], data[2]},
		freq.SVCPlayerIndexDelta, freq.SVCPlayerInfoPingLo, freq.SVCPlayerInfoPingHi)
}

func (e *Encoder) encodeSVCUpdatePL(
	enc *rangeenc.Encoder,
	ctx *encodingContext,
	data []byte,
) error {
	delta := data[0] - ctx.lastPLPlayerIndex
	ctx.lastPLPlayerIndex = data[0]
	return e.encodeRows(enc, []byte{delta, data[1]},
		freq.SVCPlayerIndexDelta, freq.SVCUpdatePLPacketLossByte)
}

func (e *Encoder) encodeAdditionalPlayer(
	enc *rangeenc.Encoder,
	plan additionalPlayerPlan,
) error {
	record := &plan.player.record
	msec := record[wire.PlayerMsecOffset]
	if plan.shortcut {
		msec |= wire.PlayerMsecShortcut
	}
	if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoMsec, msec); err != nil {
		return err
	}
	if plan.shortcut {
		return nil
	}
	if err := e.encodeRows(enc, []byte{
		plan.originMask ^ plan.base[wire.PlayerOriginMaskOffset],
		plan.angleMoveMask ^ plan.base[wire.PlayerAngleMoveMaskOffset],
		plan.stateMask ^ plan.base[wire.PlayerStateMaskOffset],
		plan.velocityMask ^ plan.base[wire.PlayerMotionMaskOffset],
	}, playerMaskDeltaRows[:]...); err != nil {
		return err
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

	angleMoveDeltas := [4]packed.WordDelta{
		plan.angleAccumulatorDelta[0],
		plan.angleAccumulatorDelta[1],
		plan.moveDelta[0],
		plan.moveDelta[1],
	}
	for i, rows := range playerAngleMoveDeltaRows {
		if err := e.encodeWordDelta(enc, angleMoveDeltas[i], rows.low, rows.high); err != nil {
			return err
		}
	}
	if err := e.encodeWordDelta(enc, plan.rollDelta, playerRollDeltaRows.low, playerRollDeltaRows.high); err != nil {
		return err
	}
	if plan.stateMask&wire.PlayerButtonsXOR != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoButtonsXOR, plan.buttonsXOR); err != nil {
			return err
		}
	}
	if plan.stateMask&wire.PlayerImpulseSet != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoImpulseSet, record[wire.PlayerImpulseOffset]); err != nil {
			return err
		}
	}
	if plan.stateMask&wire.PlayerCommandMsecDelta != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoCommandMsecDelta, plan.commandMsecDelta); err != nil {
			return err
		}
	}
	if plan.stateMask&wire.PlayerModelRemap != 0 {
		remapIndex, _ := e.state.ModelRemapIndex(record[wire.PlayerModelOffset])
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoModelRemapIndex, remapIndex); err != nil {
			return err
		}
	}
	if plan.stateMask&wire.PlayerSkinNumSet != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoSkinNumSet, record[wire.PlayerSkinNumOffset]); err != nil {
			return err
		}
	}
	if plan.stateMask&wire.PlayerEffectsXOR != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoEffectsXOR, plan.effectsXOR); err != nil {
			return err
		}
	}

	for i, rows := range playerVelocityDeltaRows {
		if err := e.encodeWordDelta(enc, plan.velocityAccumulatorDelta[i], rows.low, rows.high); err != nil {
			return err
		}
	}
	if plan.velocityMask&wire.PlayerWeaponFrameDelta != 0 {
		if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerInfoWeaponFrameDelta, plan.weaponFrameDelta); err != nil {
			return err
		}
	}
	return nil
}

type additionalPlayerPlan struct {
	player                   additionalPlayer
	base                     state.PlayerRecordBytes
	shortcut                 bool
	originMask               byte
	angleMoveMask            byte
	stateMask                byte
	velocityMask             byte
	originDelta              [3]packed.WordDelta
	angleAccumulatorDelta    [2]packed.WordDelta
	moveDelta                [2]packed.WordDelta
	rollDelta                packed.WordDelta
	velocityAccumulatorDelta [3]packed.WordDelta
	frameDelta               byte
	buttonsXOR               byte
	commandMsecDelta         byte
	effectsXOR               byte
	weaponFrameDelta         byte
	result                   state.PlayerRecord
}

func (e *Encoder) planAdditionalPlayer(
	primary primaryPlayer,
	base packetBase,
	player additionalPlayer,
) additionalPlayerPlan {
	baseRecord := basePlayerInfoRecord(
		base.players,
		player.player,
		unsignedCoordinates(primary.origin),
		e.state.PlayerModelIndex,
	)
	baseBytes := state.PlayerRecordBytesLE(baseRecord)
	target := &player.record
	shortcutBytes := baseBytes
	shortcutBytes[wire.PlayerMsecOffset] = target[wire.PlayerMsecOffset]
	// The original compressor compares the unmasked wire msec, then only the
	// 31 visible record bytes from origin through weapon frame. A wire value
	// such as 0xff can therefore select the shortcut even when its normalized
	// 0x7f value equals the history record's msec.
	visibleState := target[wire.PlayerOriginOffset:wire.PlayerVisibleStateEnd]
	baseVisibleState := baseBytes[wire.PlayerOriginOffset:wire.PlayerVisibleStateEnd]
	if e.qwdInput && player.wireMsec > baseBytes[wire.PlayerMsecOffset] && bytes.Equal(visibleState, baseVisibleState) {
		return additionalPlayerPlan{
			player:   player,
			base:     baseBytes,
			shortcut: true,
			result:   state.PlayerRecordFromBytesLE(shortcutBytes),
		}
	}
	shortcutFlags, shortcutCommandFlags := buildPlayerInfoFlags(&shortcutBytes, e.state.PlayerModelIndex)
	if !e.qwdInput && target[wire.PlayerMsecOffset] > baseBytes[wire.PlayerMsecOffset] &&
		shortcutFlags == player.flags && shortcutCommandFlags == player.commandFlags &&
		bytes.Equal(
			appendPlayerInfoRecord(nil, player.player, shortcutFlags, shortcutCommandFlags, &shortcutBytes),
			appendPlayerInfoRecord(nil, player.player, player.flags, player.commandFlags, target),
		) {
		return additionalPlayerPlan{
			player:   player,
			base:     baseBytes,
			shortcut: true,
			result:   state.PlayerRecordFromBytesLE(shortcutBytes),
		}
	}

	scale := playerPredictionScale(
		baseBytes[wire.PlayerCommandMsecOffset],
		target[wire.PlayerMsecOffset],
		baseBytes[wire.PlayerMsecOffset],
		base.predictionScale,
	)
	predictedOrigin := predictedPlayerOrigin(&baseBytes, scale)
	var velocity [3]int16
	for axis, field := range wire.PlayerVelocityFields {
		velocity[axis] = int16(state.PlayerRecordUint16(target, field.RecordOffset))
	}
	velocityAccumulators, accumulatorDeltas := playerVelocityAccumulatorDeltas(&baseBytes, velocity)
	plan := additionalPlayerPlan{
		player:                   player,
		base:                     baseBytes,
		velocityAccumulatorDelta: accumulatorDeltas,
		frameDelta:               target[wire.PlayerFrameOffset] - baseBytes[wire.PlayerFrameOffset],
		buttonsXOR:               target[wire.PlayerButtonsOffset] ^ baseBytes[wire.PlayerButtonsOffset],
		commandMsecDelta:         target[wire.PlayerCommandMsecOffset] - baseBytes[wire.PlayerCommandMsecOffset],
		effectsXOR:               target[wire.PlayerEffectsOffset] ^ baseBytes[wire.PlayerEffectsOffset],
		weaponFrameDelta:         target[wire.PlayerWeaponFrameOffset] - baseBytes[wire.PlayerWeaponFrameOffset],
	}
	for axis := range plan.originDelta {
		origin := int16(state.PlayerRecordUint16(target, wire.PlayerOriginOffset+axis*2))
		plan.originDelta[axis] = packed.SplitWordDelta(origin, predictedOrigin[axis])
	}
	for i, off := range []int{wire.PlayerAngleOffset, wire.PlayerAngleOffset + 2} {
		baseValue := int16(binary.LittleEndian.Uint16(baseBytes[off : off+2]))
		targetValue := int16(binary.LittleEndian.Uint16(target[off : off+2]))
		newAccumulator := int16(uint16(targetValue) - uint16(baseValue))
		accumulatorOff := wire.PlayerAngleAccumulatorOffset + i*2
		baseAccumulator := int16(binary.LittleEndian.Uint16(baseBytes[accumulatorOff : accumulatorOff+2]))
		plan.angleAccumulatorDelta[i] = packed.SplitWordDelta(newAccumulator, baseAccumulator)
	}
	for i, off := range []int{wire.PlayerMoveOffset, wire.PlayerMoveOffset + 2} {
		plan.moveDelta[i] = packed.SplitWordDelta(
			int16(binary.LittleEndian.Uint16(target[off:off+2])),
			int16(binary.LittleEndian.Uint16(baseBytes[off:off+2])),
		)
	}
	plan.rollDelta = packed.SplitWordDelta(
		int16(state.PlayerRecordUint16(target, wire.PlayerRollOffset)),
		int16(state.PlayerRecordUint16(&baseBytes, wire.PlayerRollOffset)),
	)
	for axis, delta := range plan.originDelta {
		setWordDeltaMask(&plan.originMask, delta, axis)
	}
	if plan.frameDelta != 0 {
		plan.originMask |= wire.PlayerFrameDelta
	}
	for i, delta := range plan.angleAccumulatorDelta {
		setWordDeltaMask(&plan.angleMoveMask, delta, i)
	}
	for i, delta := range plan.moveDelta {
		setWordDeltaMask(&plan.angleMoveMask, delta, len(plan.angleAccumulatorDelta)+i)
	}
	setWordDeltaMask(&plan.stateMask, plan.rollDelta, 0)
	if plan.buttonsXOR != 0 {
		plan.stateMask |= wire.PlayerButtonsXOR
	}
	if target[wire.PlayerImpulseOffset] != baseBytes[wire.PlayerImpulseOffset] {
		plan.stateMask |= wire.PlayerImpulseSet
	}
	if plan.commandMsecDelta != 0 {
		plan.stateMask |= wire.PlayerCommandMsecDelta
	}
	if target[wire.PlayerModelOffset] != baseBytes[wire.PlayerModelOffset] {
		plan.stateMask |= wire.PlayerModelRemap
	}
	if target[wire.PlayerSkinNumOffset] != baseBytes[wire.PlayerSkinNumOffset] {
		plan.stateMask |= wire.PlayerSkinNumSet
	}
	if plan.effectsXOR != 0 {
		plan.stateMask |= wire.PlayerEffectsXOR
	}
	for axis, delta := range plan.velocityAccumulatorDelta {
		setWordDeltaMask(&plan.velocityMask, delta, axis)
	}
	if plan.weaponFrameDelta != 0 {
		plan.velocityMask |= wire.PlayerWeaponFrameDelta
	}
	if player.flags&protocol.PFDead != 0 {
		plan.velocityMask |= wire.PlayerDead
	}

	resultBytes := baseBytes
	resultBytes[wire.PlayerOriginMaskOffset] = plan.originMask & wire.PlayerOriginHistoryMask
	resultBytes[wire.PlayerAngleMoveMaskOffset] = plan.angleMoveMask
	resultBytes[wire.PlayerStateMaskOffset] = plan.stateMask & wire.PlayerStateHistoryMask
	resultBytes[wire.PlayerMotionMaskOffset] = plan.velocityMask & wire.PlayerMotionHistoryMask
	copy(resultBytes[wire.PlayerOriginOffset:wire.PlayerVisibleStateEnd], visibleState)
	resultBytes[wire.PlayerMsecOffset] = target[wire.PlayerMsecOffset]
	for i, off := range []int{wire.PlayerAngleAccumulatorOffset, wire.PlayerAngleAccumulatorOffset + 2} {
		baseValueOff := wire.PlayerAngleOffset + i*2
		baseValue := int16(binary.LittleEndian.Uint16(baseBytes[baseValueOff : baseValueOff+2]))
		targetValue := int16(binary.LittleEndian.Uint16(target[baseValueOff : baseValueOff+2]))
		binary.LittleEndian.PutUint16(resultBytes[off:off+2], uint16(int16(uint16(targetValue)-uint16(baseValue))))
	}
	for axis := range velocityAccumulators {
		accumulatorOffset := wire.PlayerVelocityAccumulatorOffset + axis*2
		state.SetPlayerRecordUint16(&resultBytes, accumulatorOffset, uint16(velocityAccumulators[axis]))
	}
	plan.result = state.PlayerRecordFromBytesLE(resultBytes)
	return plan
}
