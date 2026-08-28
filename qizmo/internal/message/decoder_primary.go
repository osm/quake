package message

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/internal/wire"
	"github.com/osm/quake/qizmo/packed"
	"github.com/osm/quake/qizmo/state"
)

const (
	primaryWeaponFrameFlag = byte(protocol.PFWeaponFrame >> 8)
	primaryDeadFlag        = byte(protocol.PFDead >> 8)
)

func (d *packetDecoder) loadPlayerInfoBase(
	referenceSequence uint32,
) (state.PlayerRecord, []state.PlayerRecord, error) {
	st := d.state

	if _, ok := st.EntitySnapshot(referenceSequence); !ok {
		return state.PlayerRecord{}, nil, errDroppedPacket
	}

	base, found, err := st.FindPlayerBaseline(referenceSequence)
	if err != nil {
		return state.PlayerRecord{}, nil, errDroppedPacket
	}

	players, ok := st.PlayerSnapshot(referenceSequence)
	if !ok {
		return state.PlayerRecord{}, nil, errDroppedPacket
	}

	if !found {
		return st.DefaultPlayer(), players, nil
	}

	return base, players, nil
}

func (d *packetDecoder) decodeSVCPlayerInfo() ([]byte, error) {
	record, predictionScale, err := d.decodePrimaryPlayerBase()
	if err != nil {
		return nil, err
	}
	if err := d.decodePrimaryPlayerOrigin(&record, predictionScale); err != nil {
		return nil, err
	}
	flagsLow, err := d.decodePrimaryPlayerState(&record)
	if err != nil {
		return nil, err
	}
	motionFlags, flagsHigh, err := d.decodePrimaryPlayerMotion(&record)
	if err != nil {
		return nil, err
	}
	flagsLow |= motionFlags

	d.state.CurrentPlayers = append(d.state.CurrentPlayers, record)
	return appendPrimaryPlayer(nil, d.state.PlayerIndex, flagsLow, flagsHigh, record), nil
}

func (d *packetDecoder) decodePrimaryPlayerBase() (state.PlayerRecord, int, error) {
	referenceDistance, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerInfoBackReference)
	if err != nil {
		return state.PlayerRecord{}, 0, err
	}
	referenceSequence := d.state.Sequence() - uint32(referenceDistance)
	predictionScale := int(d.state.CommandScale(referenceSequence)) * int(referenceDistance)
	record := d.state.DefaultPlayer()
	if referenceDistance != 0 {
		d.state.SetPacketBase(referenceSequence)
		base, players, err := d.loadPlayerInfoBase(referenceSequence)
		if err != nil {
			return state.PlayerRecord{}, 0, err
		}
		record = base
		d.basePlayers = players
	}

	for field, row := range primaryPlayerMaskXORRows {
		delta, err := d.rd.DecodeFreqByte(d.ft, row)
		if err != nil {
			return state.PlayerRecord{}, 0, err
		}
		switch field {
		case 0:
			state.SetPlayerOriginMask(&record, state.PlayerOriginMask(record)^delta)
		case 1:
			state.SetPlayerStateMask(&record, state.PlayerStateMask(record)^delta)
		case 2:
			state.SetPlayerMotionMask(&record, state.PlayerMotionMask(record)^delta)
		}
	}
	return record, predictionScale, nil
}

func (d *packetDecoder) decodePrimaryPlayerOrigin(record *state.PlayerRecord, predictionScale int) error {
	origin := state.PlayerOrigin(*record)
	velocity := state.PlayerVelocity(*record)
	for axis := range origin {
		origin[axis] += uint16(packed.Scaled16(velocity[axis], predictionScale))
	}

	originMask := state.PlayerOriginMask(*record)
	for axis, rows := range playerOriginDeltaRows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(d.rd, d.ft, uint16(originMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		origin[axis] += uint16(delta)
	}
	if originMask&wire.PlayerFrameDelta != 0 {
		delta, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerInfoFrameDelta)
		if err != nil {
			return err
		}
		state.SetPlayerFrame(record, state.PlayerFrame(*record)+delta)
	}
	state.SetPlayerOrigin(record, origin)
	state.SetPlayerOriginMask(record, originMask&wire.PlayerOriginHistoryMask)
	return nil
}

func (d *packetDecoder) decodePrimaryPlayerState(record *state.PlayerRecord) (byte, error) {
	stateMask := state.PlayerStateMask(*record)
	var flags byte

	model := state.PlayerModel(*record)
	if stateMask&wire.PlayerModelRemap != 0 {
		index, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerInfoModelRemapIndex)
		if err != nil {
			return 0, err
		}
		model = d.state.ModelForRemapIndex(index)
		state.SetPlayerModel(record, model)
	}
	if model != d.state.PlayerModelIndex {
		flags |= protocol.PFModel
	}

	skinNum := state.PlayerSkinNum(*record)
	if stateMask&wire.PlayerSkinNumSet != 0 {
		value, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerInfoSkinNumSet)
		if err != nil {
			return 0, err
		}
		skinNum = value
		state.SetPlayerSkinNum(record, skinNum)
	}
	if skinNum != 0 {
		flags |= protocol.PFSkinNum
	}

	effects := state.PlayerEffects(*record)
	if stateMask&wire.PlayerEffectsXOR != 0 {
		value, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerInfoEffectsXOR)
		if err != nil {
			return 0, err
		}
		effects ^= value
		state.SetPlayerEffects(record, effects)
	}
	if effects != 0 {
		flags |= protocol.PFEffects
	}

	state.SetPlayerStateMask(record, stateMask&wire.PlayerStateHistoryMask)
	return flags, nil
}

func (d *packetDecoder) decodePrimaryPlayerMotion(record *state.PlayerRecord) (byte, byte, error) {
	motionMask := state.PlayerMotionMask(*record)
	velocity := state.PlayerVelocity(*record)
	accumulator := state.PlayerVelocityAccumulator(*record)
	var flags byte

	for axis, rows := range playerVelocityDeltaRows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(d.rd, d.ft, uint16(motionMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return 0, 0, err
		}
		accumulator[axis] += delta
		velocity[axis] += accumulator[axis]
		if velocity[axis] != 0 {
			flags |= byte(protocol.PFVelocity1 << axis)
		}
	}
	state.SetPlayerVelocity(record, velocity)
	state.SetPlayerVelocityAccumulator(record, accumulator)

	weaponFrame := state.PlayerWeaponFrame(*record)
	if motionMask&wire.PlayerWeaponFrameDelta != 0 {
		delta, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerInfoWeaponFrameDelta)
		if err != nil {
			return 0, 0, err
		}
		weaponFrame += delta
		state.SetPlayerWeaponFrame(record, weaponFrame)
	}
	state.SetPlayerMotionMask(record, motionMask&wire.PlayerMotionHistoryMask)

	var flagsHigh byte
	if weaponFrame != 0 {
		flagsHigh |= primaryWeaponFrameFlag
	}
	if motionMask&wire.PlayerDead != 0 {
		flagsHigh |= primaryDeadFlag
	}
	return flags, flagsHigh, nil
}

func appendPrimaryPlayer(
	out []byte,
	player byte,
	flagsLow byte,
	flagsHigh byte,
	record state.PlayerRecord,
) []byte {
	out = append(out, player, flagsLow, flagsHigh)
	for _, value := range state.PlayerOrigin(record) {
		out = appendUint16LE(out, value)
	}
	out = append(out, state.PlayerFrame(record))

	velocity := state.PlayerVelocity(record)
	if flagsLow&protocol.PFVelocity1 != 0 {
		out = appendUint16LE(out, uint16(velocity[0]))
	}
	if flagsLow&protocol.PFVelocity2 != 0 {
		out = appendUint16LE(out, uint16(velocity[1]))
	}
	if flagsLow&protocol.PFVelocity3 != 0 {
		out = appendUint16LE(out, uint16(velocity[2]))
	}
	if flagsLow&protocol.PFModel != 0 {
		out = append(out, state.PlayerModel(record))
	}
	if flagsLow&protocol.PFSkinNum != 0 {
		out = append(out, state.PlayerSkinNum(record))
	}
	if flagsLow&protocol.PFEffects != 0 {
		out = append(out, state.PlayerEffects(record))
	}
	if flagsHigh&primaryWeaponFrameFlag != 0 {
		out = append(out, state.PlayerWeaponFrame(record))
	}
	return out
}
