package codec

import (
	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
)

// Player/entity motion follows the same sequence on both sides: choose models
// from old residual history, predict movement, code or restore the residual,
// then commit the new history. Delta means movement from the last value;
// residual means the difference between that movement and its prediction.
// All delta/residual additions and subtractions wrap at the field width.
// For example, previous value 100, predicted movement 4, and new value 105
// produce residual 1 (zigzag symbol 2); decoding restores 100 + 4 + 1.
//
// Scale the previous movement for changing update intervals. The v1 predictor
// uses int64 intermediates and rounds halves away from zero before narrowing.
func predictMotion16(
	previousDelta uint16,
	interval, previousInterval uint32,
) uint16 {
	if previousInterval == 0 {
		return previousDelta
	}
	scaled := int64(int16(previousDelta)) * int64(interval)
	divisor := int64(previousInterval)
	if scaled >= 0 {
		scaled += divisor / 2
	} else {
		scaled -= divisor / 2
	}
	return uint16(int16(scaled / divisor))
}

func predictMotion8(
	previousDelta byte,
	interval, previousInterval uint32,
) byte {
	if previousInterval == 0 {
		return previousDelta
	}
	scaled := int64(int8(previousDelta)) * int64(interval)
	divisor := int64(previousInterval)
	if scaled >= 0 {
		scaled += divisor / 2
	} else {
		scaled -= divisor / 2
	}
	return byte(int8(scaled / divisor))
}

// Motion words encode the high byte first, unlike putWord. The model for the
// low byte depends on whether that high byte was zero. Preserve this wire order.
func putMotionWord(
	sink symbolSink,
	models modelPair,
	lowHighNonzeroModel frequency.Model,
	value uint16,
) error {
	high := byte(value >> 8)
	if err := sink.Put(models[1], high); err != nil {
		return err
	}
	lowModel := models[0]
	if high != 0 {
		lowModel = lowHighNonzeroModel
	}
	return sink.Put(lowModel, byte(value))
}

func decodeMotionWord(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	wordModels modelPair,
	lowHighNonzeroModel frequency.Model,
) (uint16, error) {
	high, err := decodeSymbol(decoder, models, wordModels[1])
	if err != nil {
		return 0, err
	}
	lowModel := wordModels[0]
	if high != 0 {
		lowModel = lowHighNonzeroModel
	}
	low, err := decodeSymbol(decoder, models, lowModel)
	if err != nil {
		return 0, err
	}
	return uint16(low) | uint16(high)<<8, nil
}
