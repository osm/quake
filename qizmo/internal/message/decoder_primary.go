package message

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
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
	velocityXY := record[7]
	velocityZ := record[8]
	record[1] = packed.AddLow16(record[1], packed.Scaled16(int16(velocityXY), predictionScale))
	record[1] = packed.AddHigh16(record[1], packed.Scaled16(int16(velocityXY>>16), predictionScale))
	record[2] = packed.AddLow16(record[2], packed.Scaled16(int16(velocityZ), predictionScale))

	originMask := state.PlayerOriginMask(*record)
	for axis, rows := range playerOriginDeltaRows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(d.rd, d.ft, uint16(originMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return err
		}
		switch axis {
		case 0:
			record[1] = packed.AddLow16(record[1], delta)
		case 1:
			record[1] = packed.AddHigh16(record[1], delta)
		case 2:
			record[2] = packed.AddLow16(record[2], delta)
		}
	}
	if originMask&0x40 != 0 {
		delta, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerInfoFrameDelta)
		if err != nil {
			return err
		}
		state.SetPlayerFrame(record, state.PlayerFrame(*record)+delta)
	}
	state.SetPlayerOriginMask(record, originMask&0xbf)
	return nil
}

func (d *packetDecoder) decodePrimaryPlayerState(record *state.PlayerRecord) (byte, error) {
	stateMask := state.PlayerStateMask(*record)
	var flags byte

	model := state.PlayerModel(*record)
	if stateMask&0x20 != 0 {
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
	if stateMask&0x40 != 0 {
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
	if stateMask&0x80 != 0 {
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

	state.SetPlayerStateMask(record, stateMask&0x97)
	return flags, nil
}

func (d *packetDecoder) decodePrimaryPlayerMotion(record *state.PlayerRecord) (byte, byte, error) {
	motionMask := state.PlayerMotionMask(*record)
	velocityXY := record[7]
	velocityZ := record[8]
	accumulatorXY := record[10]
	accumulatorZ := record[11]
	var flags byte

	for axis, rows := range playerVelocityDeltaRows {
		lowMask := uint16(1 << uint(axis*2))
		delta, err := decodeMaskedWordDelta(d.rd, d.ft, uint16(motionMask), lowMask, lowMask<<1, rows)
		if err != nil {
			return 0, 0, err
		}
		var velocity int16
		switch axis {
		case 0:
			accumulatorXY = packed.AddLow16(accumulatorXY, delta)
			velocity = int16(velocityXY) + int16(accumulatorXY)
			velocityXY = packed.SetLow16(velocityXY, velocity)
		case 1:
			accumulatorXY = packed.AddHigh16(accumulatorXY, delta)
			velocity = int16(velocityXY>>16) + int16(accumulatorXY>>16)
			velocityXY = packed.SetHigh16(velocityXY, velocity)
		case 2:
			accumulatorZ = packed.AddLow16(accumulatorZ, delta)
			velocity = int16(velocityZ) + int16(accumulatorZ)
			velocityZ = packed.SetLow16(velocityZ, velocity)
		}
		if velocity != 0 {
			flags |= byte(protocol.PFVelocity1 << axis)
		}
	}
	record[7] = velocityXY
	record[10] = accumulatorXY
	record[11] = accumulatorZ

	weaponFrame := byte(velocityZ >> 16)
	if motionMask&0x40 != 0 {
		delta, err := d.rd.DecodeFreqByte(d.ft, freq.SVCPlayerInfoWeaponFrameDelta)
		if err != nil {
			return 0, 0, err
		}
		weaponFrame += delta
		velocityZ = velocityZ&^0x00ff0000 | uint32(weaponFrame)<<16
	}
	record[8] = velocityZ
	state.SetPlayerMotionMask(record, motionMask&0xbf)

	var flagsHigh byte
	if weaponFrame != 0 {
		flagsHigh |= primaryWeaponFrameFlag
	}
	if motionMask&0x80 != 0 {
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
	out = appendUint32LE(out, record[1])
	out = append(out, byte(record[2]), byte(record[2]>>8), byte(record[2]>>16))

	if flagsLow&protocol.PFVelocity1 != 0 {
		out = appendUint16LE(out, uint16(record[7]))
	}
	if flagsLow&protocol.PFVelocity2 != 0 {
		out = appendUint16LE(out, uint16(record[7]>>16))
	}
	if flagsLow&protocol.PFVelocity3 != 0 {
		out = appendUint16LE(out, uint16(record[8]))
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
