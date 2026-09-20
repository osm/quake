package codec

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
)

type rangeFixedOperation struct {
	opcode  byte
	payload []byte
}

var updateStatLongModels = [...]frequency.Model{
	frequency.ModelUpdateStatLongDeltaByte0,
	frequency.ModelUpdateStatLongDeltaByte1,
	frequency.ModelUpdateStatLongDeltaByte2,
	frequency.ModelUpdateStatLongDeltaByte3,
}

var muzzleFlashModels = modelPair{
	frequency.ModelMuzzleFlashLoDelta,
	frequency.ModelMuzzleFlashHiDelta,
}

func fixedPayloadSize(opcode byte) (int, bool) {
	switch opcode {
	case protocol.SVCUpdateStat,
		protocol.SVCMuzzleFlash,
		protocol.SVCUpdatePL:
		return 2, true
	case protocol.SVCUpdateFrags, protocol.SVCUpdatePing:
		return 3, true
	case protocol.SVCUpdateStatLong:
		return 5, true
	case protocol.SVCSmallKick, protocol.SVCBigKick:
		return 0, true
	default:
		return 0, false
	}
}

func operationModelAfterFixed(opcode byte) (frequency.Model, bool) {
	switch opcode {
	case protocol.SVCUpdateStat:
		return frequency.ModelOperationKindAfterUpdateStat, true
	case protocol.SVCUpdateFrags:
		return frequency.ModelOperationKindAfterUpdateFrags, true
	case protocol.SVCSmallKick, protocol.SVCBigKick:
		return frequency.ModelOperationKindAfterFixed, true
	case protocol.SVCUpdatePing:
		return frequency.ModelOperationKindAfterUpdatePing, true
	case protocol.SVCUpdateStatLong:
		return frequency.ModelOperationKindAfterUpdateStatLong, true
	case protocol.SVCMuzzleFlash:
		return frequency.ModelOperationKindAfterMuzzleFlash, true
	case protocol.SVCUpdatePL:
		return frequency.ModelOperationKindAfterUpdatePL, true
	default:
		return 0, false
	}
}

func newFixedOperation(raw []byte) *rangeFixedOperation {
	if len(raw) == 0 {
		return nil
	}
	size, ok := fixedPayloadSize(raw[0])
	if !ok || len(raw) != size+1 {
		return nil
	}
	return &rangeFixedOperation{opcode: raw[0], payload: raw[1:]}
}

func encodeFixedOperation(
	sink symbolSink,
	state *rangeCodecState,
	operation rangeFixedOperation,
) error {
	switch operation.opcode {
	case protocol.SVCUpdateStat:
		return encodeUpdateStat(sink, state, operation.payload)
	case protocol.SVCUpdateStatLong:
		return encodeUpdateStatLong(sink, state, operation.payload)
	case protocol.SVCUpdateFrags, protocol.SVCUpdatePing, protocol.SVCUpdatePL:
		return encodePlayerUpdate(sink, state, operation)
	case protocol.SVCMuzzleFlash:
		return encodeMuzzleFlash(sink, state, operation.payload)
	case protocol.SVCSmallKick, protocol.SVCBigKick:
		return nil
	default:
		return fmt.Errorf("unsupported fixed operation %d", operation.opcode)
	}
}

func decodeFixedOperation(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	opcode byte,
) ([]byte, error) {
	size, ok := fixedPayloadSize(opcode)
	if !ok {
		return nil, fmt.Errorf("unsupported fixed operation %d", opcode)
	}
	out := make([]byte, size+1)
	out[0] = opcode
	var decodeErr error
	switch opcode {
	case protocol.SVCUpdateStat:
		decodeErr = decodeUpdateStat(decoder, models, state, out)
	case protocol.SVCUpdateStatLong:
		decodeErr = decodeUpdateStatLong(decoder, models, state, out)
	case protocol.SVCUpdateFrags, protocol.SVCUpdatePing, protocol.SVCUpdatePL:
		decodeErr = decodePlayerUpdate(decoder, models, state, opcode, out)
	case protocol.SVCMuzzleFlash:
		decodeErr = decodeMuzzleFlash(decoder, models, state, out)
	case protocol.SVCSmallKick, protocol.SVCBigKick:
	}
	if decodeErr != nil {
		return nil, decodeErr
	}
	return out, nil
}

func encodeUpdateStat(
	sink symbolSink,
	state *rangeCodecState,
	payload []byte,
) error {
	index := payload[0]
	target := state.currentCommandTarget
	indexSymbol := zigzag8(index - state.previousStatIndex[target])
	if err := sink.Put(frequency.ModelUpdateStatIndexDelta, indexSymbol); err != nil {
		return err
	}
	valueSymbol := zigzag8(payload[1] - state.previousStatValues[target][index])
	if err := sink.Put(frequency.ModelUpdateStatValueDelta, valueSymbol); err != nil {
		return err
	}
	state.previousStatIndex[target] = index
	state.previousStatValues[target][index] = payload[1]
	return nil
}

func decodeUpdateStat(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	out []byte,
) error {
	target := state.currentCommandTarget
	indexSymbol, err := decodeSymbol(decoder, models, frequency.ModelUpdateStatIndexDelta)
	if err != nil {
		return err
	}
	index := state.previousStatIndex[target] + unzigzag8(indexSymbol)
	valueSymbol, err := decodeSymbol(decoder, models, frequency.ModelUpdateStatValueDelta)
	if err != nil {
		return err
	}
	value := state.previousStatValues[target][index] + unzigzag8(valueSymbol)
	out[1], out[2] = index, value
	state.previousStatIndex[target] = index
	state.previousStatValues[target][index] = value
	return nil
}

func encodeUpdateStatLong(
	sink symbolSink,
	state *rangeCodecState,
	payload []byte,
) error {
	index := payload[0]
	target := state.currentCommandTarget
	value := binary.LittleEndian.Uint32(payload[1:])
	valueSymbol := zigzag32(value - state.previousStatLongValues[target][index])
	indexSymbol := zigzag8(index - state.previousStatLongIndex[target])
	if err := sink.Put(frequency.ModelUpdateStatLongIndexDelta, indexSymbol); err != nil {
		return err
	}
	for i, model := range updateStatLongModels {
		if err := sink.Put(model, byte(valueSymbol>>uint(8*i))); err != nil {
			return err
		}
	}
	state.previousStatLongIndex[target] = index
	state.previousStatLongValues[target][index] = value
	return nil
}

func decodeUpdateStatLong(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	out []byte,
) error {
	target := state.currentCommandTarget
	indexSymbol, err := decodeSymbol(
		decoder, models, frequency.ModelUpdateStatLongIndexDelta,
	)
	if err != nil {
		return err
	}
	index := state.previousStatLongIndex[target] + unzigzag8(indexSymbol)
	var valueSymbol uint32
	for i, model := range updateStatLongModels {
		byteSymbol, err := decodeSymbol(decoder, models, model)
		if err != nil {
			return err
		}
		valueSymbol |= uint32(byteSymbol) << uint(8*i)
	}
	value := state.previousStatLongValues[target][index] + unzigzag32(valueSymbol)
	out[1] = index
	binary.LittleEndian.PutUint32(out[2:], value)
	state.previousStatLongIndex[target] = index
	state.previousStatLongValues[target][index] = value
	return nil
}

func encodePlayerUpdate(
	sink symbolSink,
	state *rangeCodecState,
	operation rangeFixedOperation,
) error {
	player := operation.payload[0]
	if player >= rangePlayerSlots {
		return fmt.Errorf("invalid player index %d", player)
	}
	playerSymbol := zigzag8(player - state.previousFixedPlayer)
	if err := sink.Put(frequency.ModelFixedPlayerIndexDelta, playerSymbol); err != nil {
		return err
	}
	state.previousFixedPlayer = player
	if operation.opcode == protocol.SVCUpdatePL {
		symbol := zigzag8(operation.payload[1] - state.previousPacketLoss[player])
		if err := sink.Put(frequency.ModelUpdatePLDelta, symbol); err != nil {
			return err
		}
		state.previousPacketLoss[player] = operation.payload[1]
		return nil
	}
	return encodePlayerWord(sink, state, operation, player)
}

func decodePlayerUpdate(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	opcode byte,
	out []byte,
) error {
	playerSymbol, err := decodeSymbol(
		decoder, models, frequency.ModelFixedPlayerIndexDelta,
	)
	if err != nil {
		return err
	}
	player := state.previousFixedPlayer + unzigzag8(playerSymbol)
	if player >= rangePlayerSlots {
		return fmt.Errorf("invalid player index %d", player)
	}
	out[1] = player
	state.previousFixedPlayer = player
	if opcode == protocol.SVCUpdatePL {
		symbol, err := decodeSymbol(decoder, models, frequency.ModelUpdatePLDelta)
		if err != nil {
			return err
		}
		value := state.previousPacketLoss[player] + unzigzag8(symbol)
		out[2] = value
		state.previousPacketLoss[player] = value
		return nil
	}
	return decodePlayerWord(decoder, models, state, opcode, player, out)
}

func encodePlayerWord(
	sink symbolSink,
	state *rangeCodecState,
	operation rangeFixedOperation,
	player byte,
) error {
	value := binary.LittleEndian.Uint16(operation.payload[1:])
	previous, models := playerWordContext(state, operation.opcode, player)
	symbol := zigzag16(value - *previous)
	if err := putWord(sink, models, symbol); err != nil {
		return err
	}
	*previous = value
	return nil
}

func decodePlayerWord(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	opcode, player byte,
	out []byte,
) error {
	previous, wordModels := playerWordContext(state, opcode, player)
	symbol, err := decodeWord(decoder, models, wordModels)
	if err != nil {
		return err
	}
	value := *previous + unzigzag16(symbol)
	binary.LittleEndian.PutUint16(out[2:], value)
	*previous = value
	return nil
}

func playerWordContext(
	state *rangeCodecState,
	opcode, player byte,
) (*uint16, modelPair) {
	if opcode == protocol.SVCUpdatePing {
		return &state.previousPings[player], modelPair{
			frequency.ModelUpdatePingLoDelta,
			frequency.ModelUpdatePingHiDelta,
		}
	}
	return &state.previousFrags[player], modelPair{
		frequency.ModelUpdateFragsLoDelta,
		frequency.ModelUpdateFragsHiDelta,
	}
}

func encodeMuzzleFlash(
	sink symbolSink,
	state *rangeCodecState,
	payload []byte,
) error {
	entity := binary.LittleEndian.Uint16(payload)
	symbol := zigzag16(entity - state.previousEffectEntity)
	if err := putWord(sink, muzzleFlashModels, symbol); err != nil {
		return err
	}
	state.previousEffectEntity = entity
	return nil
}

func decodeMuzzleFlash(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	out []byte,
) error {
	symbol, err := decodeWord(decoder, models, muzzleFlashModels)
	if err != nil {
		return err
	}
	entity := state.previousEffectEntity + unzigzag16(symbol)
	binary.LittleEndian.PutUint16(out[1:], entity)
	state.previousEffectEntity = entity
	return nil
}
