package codec

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
)

// Movement deltas drive prediction. Residuals before zigzag select its models.
// Select models before updating history so the decoder sees the same context.
// Origins stored in 32 bits skip residual contexts, so residual history is cleared.
type rangeEntityState struct {
	bits           byte
	flags          [3]byte
	model          byte
	frame          byte
	colorMap       byte
	skin           byte
	effects        byte
	origin         [3]uint32
	angle          [3]uint16
	originDelta    [3]uint32
	angleDelta     [3]uint16
	originResidual [3]uint32
	angleResidual  [3]uint16
	originTime     [3]uint32
	angleTime      [3]uint32
	originInterval [3]uint32
	angleInterval  [3]uint32
}

type rangeEntityFieldEncoder struct {
	sink        symbolSink
	state       *rangeEntityState
	layout      rangeEntityLayout
	coordSize   int
	angleSize   int
	currentTime uint32
	offset      int
}

type rangeEntityFieldDecoder struct {
	decoder      *rangecoder.Decoder
	models       *frequency.Models
	state        *rangeEntityState
	bits         uint16
	coordSize    int
	angleSize    int
	currentTime  uint32
	decodedLimit int
	out          []byte
}

func encodeEntityFields(
	sink symbolSink,
	state *rangeEntityState,
	layout rangeEntityLayout,
	coordSize, angleSize int,
	currentTime uint32,
) error {
	encoder := rangeEntityFieldEncoder{
		sink:        sink,
		state:       state,
		layout:      layout,
		coordSize:   coordSize,
		angleSize:   angleSize,
		currentTime: currentTime,
	}
	return encoder.encode()
}

func decodeEntityFields(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeEntityState,
	bits uint16,
	coordSize, angleSize int,
	currentTime uint32,
	decodedLimit int,
) ([]byte, error) {
	fieldDecoder := rangeEntityFieldDecoder{
		decoder:      decoder,
		models:       models,
		state:        state,
		bits:         bits,
		coordSize:    coordSize,
		angleSize:    angleSize,
		currentTime:  currentTime,
		decodedLimit: decodedLimit,
	}
	return fieldDecoder.decode()
}

func (e *rangeEntityFieldEncoder) encode() error {
	if err := e.encodeByteFields(); err != nil {
		return err
	}
	for axis := 0; axis < 3; axis++ {
		if err := e.encodeAxis(axis); err != nil {
			return err
		}
	}
	if e.offset != len(e.layout.payload) {
		return fmt.Errorf(
			"consumed %d field bytes, have %d",
			e.offset,
			len(e.layout.payload),
		)
	}
	return e.encodeTail()
}

func (d *rangeEntityFieldDecoder) decode() ([]byte, error) {
	if err := d.decodeByteFields(); err != nil {
		return nil, err
	}
	for axis := 0; axis < 3; axis++ {
		if err := d.decodeAxis(axis); err != nil {
			return nil, err
		}
	}
	if err := d.decodeTail(); err != nil {
		return nil, err
	}
	return d.out, nil
}

func (e *rangeEntityFieldEncoder) encodeByteFields() error {
	byteFields := [...]struct {
		mask  uint16
		model frequency.Model
		value *byte
		xor   bool
	}{
		{protocol.UModel, frequency.ModelEntityModelDelta, &e.state.model, false},
		{protocol.UFrame, frequency.ModelEntityFrameDelta, &e.state.frame, false},
		{protocol.UColorMap, frequency.ModelEntityColorMapDelta, &e.state.colorMap, false},
		{protocol.USkin, frequency.ModelEntitySkinDelta, &e.state.skin, false},
		{protocol.UEffects, frequency.ModelEntityEffectsXOR, &e.state.effects, true},
	}
	for _, field := range byteFields {
		if e.layout.bits&field.mask == 0 {
			continue
		}
		value := e.layout.payload[e.offset]
		symbol := value - *field.value
		if field.xor {
			symbol = value ^ *field.value
		}
		if err := e.sink.Put(field.model, symbol); err != nil {
			return err
		}
		*field.value = value
		e.offset++
	}
	return nil
}

func (d *rangeEntityFieldDecoder) decodeByteFields() error {
	byteFields := [...]struct {
		mask  uint16
		model frequency.Model
		value *byte
		xor   bool
	}{
		{protocol.UModel, frequency.ModelEntityModelDelta, &d.state.model, false},
		{protocol.UFrame, frequency.ModelEntityFrameDelta, &d.state.frame, false},
		{protocol.UColorMap, frequency.ModelEntityColorMapDelta, &d.state.colorMap, false},
		{protocol.USkin, frequency.ModelEntitySkinDelta, &d.state.skin, false},
		{protocol.UEffects, frequency.ModelEntityEffectsXOR, &d.state.effects, true},
	}
	for _, field := range byteFields {
		if d.bits&field.mask == 0 {
			continue
		}
		symbol, err := decodeSymbol(d.decoder, d.models, field.model)
		if err != nil {
			return err
		}
		value := *field.value + symbol
		if field.xor {
			value = *field.value ^ symbol
		}
		*field.value = value
		d.out = append(d.out, value)
	}
	return nil
}

func (e *rangeEntityFieldEncoder) encodeAxis(axis int) error {
	if e.layout.bits&(protocol.UOrigin1<<axis) != 0 {
		if err := e.encodeOrigin(axis); err != nil {
			return err
		}
	}
	if e.layout.bits&entityAngleMask(axis) != 0 {
		return e.encodeAngle(axis)
	}
	return nil
}

func (d *rangeEntityFieldDecoder) decodeAxis(axis int) error {
	if d.bits&(protocol.UOrigin1<<axis) != 0 {
		if err := d.decodeOrigin(axis); err != nil {
			return err
		}
	}
	if d.bits&entityAngleMask(axis) != 0 {
		return d.decodeAngle(axis)
	}
	return nil
}

func (e *rangeEntityFieldEncoder) encodeOrigin(axis int) error {
	models := entityOriginModelsFor(e.state, axis, e.coordSize)
	if e.coordSize == 2 {
		if err := e.encodeOrigin16(axis, models); err != nil {
			return err
		}
	} else if err := e.encodeOrigin32(axis, models); err != nil {
		return err
	}
	e.offset += e.coordSize
	return nil
}

func (d *rangeEntityFieldDecoder) decodeOrigin(axis int) error {
	models := entityOriginModelsFor(d.state, axis, d.coordSize)
	symbol, err := decodeMotionWord(
		d.decoder,
		d.models,
		models,
		entityOriginLowHighNonzeroModels[axis],
	)
	if err != nil {
		return err
	}
	if d.coordSize == 2 {
		d.restoreOrigin16(axis, symbol)
		return nil
	}
	return d.decodeOrigin32(axis, symbol)
}

func (e *rangeEntityFieldEncoder) encodeOrigin16(
	axis int,
	models modelPair,
) error {
	state := e.state
	value := uint32(binary.LittleEndian.Uint16(e.layout.payload[e.offset:]))
	delta := uint16(value - state.origin[axis])
	interval := e.currentTime - state.originTime[axis]
	prediction := predictMotion16(
		uint16(state.originDelta[axis]), interval, state.originInterval[axis],
	)
	residual := delta - prediction
	symbol := zigzag16(residual)

	if err := putMotionWord(
		e.sink,
		models,
		entityOriginLowHighNonzeroModels[axis],
		symbol,
	); err != nil {
		return err
	}
	state.originResidual[axis] = uint32(residual)
	state.originDelta[axis] = uint32(delta)
	state.originInterval[axis] = interval
	state.originTime[axis] = e.currentTime
	state.origin[axis] = value
	return nil
}

func (d *rangeEntityFieldDecoder) restoreOrigin16(axis int, symbol uint16) {
	state := d.state
	residual := unzigzag16(symbol)
	interval := d.currentTime - state.originTime[axis]
	prediction := predictMotion16(
		uint16(state.originDelta[axis]), interval, state.originInterval[axis],
	)
	delta := residual + prediction
	value := uint16(state.origin[axis]) + delta

	state.originResidual[axis] = uint32(residual)
	state.originDelta[axis] = uint32(delta)
	state.originInterval[axis] = interval
	state.originTime[axis] = d.currentTime
	state.origin[axis] = uint32(value)

	d.out = binary.LittleEndian.AppendUint16(d.out, value)
}

func (e *rangeEntityFieldEncoder) encodeOrigin32(
	axis int,
	models modelPair,
) error {
	// Raw coordinate bits are not distances, so they are not scaled by time.
	state := e.state
	value := binary.LittleEndian.Uint32(e.layout.payload[e.offset:])
	delta := value - state.origin[axis]
	prediction := state.originDelta[axis]
	residual := delta - prediction
	symbol := zigzag32(residual)

	// Match player origins: motion-code bytes 1, 0, then emit bytes 2, 3.
	if err := putMotionWord(
		e.sink,
		models,
		entityOriginLowHighNonzeroModels[axis],
		uint16(symbol),
	); err != nil {
		return err
	}
	if err := e.sink.Put(
		frequency.ModelEntityOriginByte2Delta, byte(symbol>>16),
	); err != nil {
		return err
	}
	if err := e.sink.Put(
		frequency.ModelEntityOriginByte3Delta, byte(symbol>>24),
	); err != nil {
		return err
	}
	state.originResidual[axis] = 0
	state.originDelta[axis] = delta
	state.origin[axis] = value
	return nil
}

func (d *rangeEntityFieldDecoder) decodeOrigin32(
	axis int,
	lowSymbol uint16,
) error {
	byte2, err := decodeSymbol(
		d.decoder, d.models, frequency.ModelEntityOriginByte2Delta,
	)
	if err != nil {
		return err
	}
	byte3, err := decodeSymbol(
		d.decoder, d.models, frequency.ModelEntityOriginByte3Delta,
	)
	if err != nil {
		return err
	}
	state := d.state
	symbol := uint32(lowSymbol) | uint32(byte2)<<16 | uint32(byte3)<<24
	residual := unzigzag32(symbol)
	prediction := state.originDelta[axis]
	delta := residual + prediction
	value := state.origin[axis] + delta

	state.originResidual[axis] = 0
	state.originDelta[axis] = delta
	state.origin[axis] = value

	d.out = binary.LittleEndian.AppendUint32(d.out, value)
	return nil
}

func (e *rangeEntityFieldEncoder) encodeAngle(axis int) error {
	models := entityAngleModelsFor(e.state, axis, e.angleSize)
	if e.angleSize == 1 {
		if err := e.encodeAngle8(axis, models[0]); err != nil {
			return err
		}
	} else if err := e.encodeAngle16(axis, models); err != nil {
		return err
	}
	e.offset += e.angleSize
	return nil
}

func (d *rangeEntityFieldDecoder) decodeAngle(axis int) error {
	models := entityAngleModelsFor(d.state, axis, d.angleSize)
	if d.angleSize == 1 {
		return d.decodeAngle8(axis, models[0])
	}
	return d.decodeAngle16(axis, models)
}

func (e *rangeEntityFieldEncoder) encodeAngle8(
	axis int,
	model frequency.Model,
) error {
	state := e.state
	value := e.layout.payload[e.offset]
	delta := value - byte(state.angle[axis])
	interval := e.currentTime - state.angleTime[axis]
	prediction := predictMotion8(
		byte(state.angleDelta[axis]), interval, state.angleInterval[axis],
	)
	residual := delta - prediction
	symbol := zigzag8(residual)

	if err := e.sink.Put(model, symbol); err != nil {
		return err
	}
	state.angleResidual[axis] = uint16(residual)
	state.angleDelta[axis] = uint16(delta)
	state.angleInterval[axis] = interval
	state.angleTime[axis] = e.currentTime
	state.angle[axis] = uint16(value)
	return nil
}

func (d *rangeEntityFieldDecoder) decodeAngle8(
	axis int,
	model frequency.Model,
) error {
	symbol, err := decodeSymbol(d.decoder, d.models, model)
	if err != nil {
		return err
	}
	state := d.state
	residual := unzigzag8(symbol)
	interval := d.currentTime - state.angleTime[axis]
	prediction := predictMotion8(
		byte(state.angleDelta[axis]), interval, state.angleInterval[axis],
	)
	delta := residual + prediction
	value := byte(state.angle[axis]) + delta

	state.angleResidual[axis] = uint16(residual)
	state.angleDelta[axis] = uint16(delta)
	state.angleInterval[axis] = interval
	state.angleTime[axis] = d.currentTime
	state.angle[axis] = uint16(value)

	d.out = append(d.out, value)
	return nil
}

func (e *rangeEntityFieldEncoder) encodeAngle16(
	axis int,
	models modelPair,
) error {
	state := e.state
	value := binary.LittleEndian.Uint16(e.layout.payload[e.offset:])
	delta := value - state.angle[axis]
	interval := e.currentTime - state.angleTime[axis]
	prediction := predictMotion16(
		state.angleDelta[axis], interval, state.angleInterval[axis],
	)
	residual := delta - prediction
	symbol := zigzag16(residual)

	if err := putMotionWord(
		e.sink,
		models,
		entityAngleLowHighNonzeroModels[axis],
		symbol,
	); err != nil {
		return err
	}
	state.angleResidual[axis] = residual
	state.angleDelta[axis] = delta
	state.angleInterval[axis] = interval
	state.angleTime[axis] = e.currentTime
	state.angle[axis] = value
	return nil
}

func (d *rangeEntityFieldDecoder) decodeAngle16(
	axis int,
	models modelPair,
) error {
	symbol, err := decodeMotionWord(
		d.decoder,
		d.models,
		models,
		entityAngleLowHighNonzeroModels[axis],
	)
	if err != nil {
		return err
	}
	state := d.state
	residual := unzigzag16(symbol)
	interval := d.currentTime - state.angleTime[axis]
	prediction := predictMotion16(
		state.angleDelta[axis], interval, state.angleInterval[axis],
	)
	delta := residual + prediction
	value := state.angle[axis] + delta

	state.angleResidual[axis] = residual
	state.angleDelta[axis] = delta
	state.angleInterval[axis] = interval
	state.angleTime[axis] = d.currentTime
	state.angle[axis] = value

	d.out = binary.LittleEndian.AppendUint16(d.out, value)
	return nil
}

func (e *rangeEntityFieldEncoder) encodeTail() error {
	if err := putUvarint(
		e.sink, frequency.ModelEntityTailLength, uint64(len(e.layout.tail)),
	); err != nil {
		return err
	}
	for _, value := range e.layout.tail {
		if err := e.sink.Put(frequency.ModelEntityTailByte, value); err != nil {
			return err
		}
	}
	return nil
}

func (d *rangeEntityFieldDecoder) decodeTail() error {
	tailLength, err := decodeUvarint(
		d.decoder, d.models, frequency.ModelEntityTailLength,
	)
	if err != nil {
		return err
	}
	if len(d.out) > d.decodedLimit ||
		tailLength > uint64(d.decodedLimit-len(d.out)) {
		return fmt.Errorf(
			"packet-entity extension tail %d exceeds decoded chunk remainder",
			tailLength,
		)
	}
	for i := uint64(0); i < tailLength; i++ {
		value, err := decodeSymbol(d.decoder, d.models, frequency.ModelEntityTailByte)
		if err != nil {
			return err
		}
		d.out = append(d.out, value)
	}
	return nil
}
