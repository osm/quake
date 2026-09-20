package codec

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
)

const rangePlayerSlots = protocol.QWMaxClients

type rangePlayerInfo struct {
	raw       []byte
	coordSize int
}

// Movement deltas drive prediction. Residuals before zigzag select its models.
// Select models before updating history so the decoder sees the same context.
// Origins stored in 32 bits skip residual contexts, so residual history is cleared.
type rangePlayerState struct {
	bits           uint16
	frame          byte
	origin         [3]uint32
	angle          [3]uint16
	model          byte
	skin           byte
	effects        byte
	weaponFrame    byte
	originDelta    [3]uint32
	angleDelta     [3]uint16
	originResidual [3]uint32
	angleResidual  [3]uint16
	originTime     [3]uint32
	angleTime      [3]uint32
	originInterval [3]uint32
	angleInterval  [3]uint32
	bitsResidual   uint16
	frameResidual  byte
}

type rangePlayerInfoEncoder struct {
	sink      symbolSink
	state     *rangeCodecState
	player    *rangePlayerState
	raw       []byte
	bits      uint16
	coordSize int
	offset    int
}

type rangePlayerInfoDecoder struct {
	decoder   *rangecoder.Decoder
	models    *frequency.Models
	state     *rangeCodecState
	player    *rangePlayerState
	bits      uint16
	coordSize int
	out       []byte
}

func validPlayerInfo(raw []byte, coordSize int) bool {
	if !validCoordSize(coordSize) {
		return false
	}
	if len(raw) < 5 || raw[0] != protocol.SVCPlayerInfo || raw[1] >= rangePlayerSlots {
		return false
	}
	bits := binary.LittleEndian.Uint16(raw[2:4])
	offset := 5
	for axis := 0; axis < 3; axis++ {
		if bits&(protocol.DFOrigin<<axis) != 0 {
			offset += coordSize
		}
	}
	for axis := 0; axis < 3; axis++ {
		if bits&(protocol.DFAngles<<axis) != 0 {
			offset += 2
		}
	}
	for _, mask := range [...]uint16{
		protocol.DFModel,
		protocol.DFSkinNum,
		protocol.DFEffects,
		protocol.DFWeaponFrame,
	} {
		if bits&mask != 0 {
			offset++
		}
	}
	return offset == len(raw)
}

func encodePlayerInfo(
	sink symbolSink,
	state *rangeCodecState,
	info rangePlayerInfo,
) error {
	if !validPlayerInfo(info.raw, info.coordSize) {
		return fmt.Errorf("invalid MVD svc_playerinfo")
	}
	encoder := rangePlayerInfoEncoder{
		sink:      sink,
		state:     state,
		raw:       info.raw,
		coordSize: info.coordSize,
		offset:    5,
	}
	return encoder.encode()
}

func decodePlayerInfo(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
) ([]byte, error) {
	fieldDecoder := rangePlayerInfoDecoder{
		decoder: decoder,
		models:  models,
		state:   state,
	}
	return fieldDecoder.decode()
}

func (e *rangePlayerInfoEncoder) encode() error {
	if err := e.encodeHeader(); err != nil {
		return err
	}
	for axis := range playerOriginModels {
		if e.bits&(protocol.DFOrigin<<axis) == 0 {
			continue
		}
		if err := e.encodeOrigin(axis); err != nil {
			return err
		}
	}
	for axis := range playerAngleModels {
		if e.bits&(protocol.DFAngles<<axis) == 0 {
			continue
		}
		if err := e.encodeAngle(axis); err != nil {
			return err
		}
	}
	return e.encodeByteFields()
}

func (d *rangePlayerInfoDecoder) decode() ([]byte, error) {
	if err := d.decodeHeader(); err != nil {
		return nil, err
	}
	for axis := range playerOriginModels {
		if d.bits&(protocol.DFOrigin<<axis) == 0 {
			continue
		}
		if err := d.decodeOrigin(axis); err != nil {
			return nil, err
		}
	}
	for axis := range playerAngleModels {
		if d.bits&(protocol.DFAngles<<axis) == 0 {
			continue
		}
		if err := d.decodeAngle(axis); err != nil {
			return nil, err
		}
	}
	if err := d.decodeByteFields(); err != nil {
		return nil, err
	}
	return d.out, nil
}

func (e *rangePlayerInfoEncoder) encodeHeader() error {
	index := e.raw[1]
	e.player = &e.state.players[index]
	e.bits = binary.LittleEndian.Uint16(e.raw[2:4])

	indexDelta := index - e.state.previousPlayerIndex
	if err := e.sink.Put(playerIndexModel(e.state), indexDelta); err != nil {
		return err
	}
	e.state.previousPlayerIndex = index
	e.state.previousPlayerIndexDelta = indexDelta
	bitsModels := playerBitsModels(e.player)
	frameModel := playerFrameModel(e.player)
	bitsXOR := e.bits ^ e.player.bits
	frameDelta := e.raw[4] - e.player.frame
	if err := e.sink.Put(bitsModels[0], byte(bitsXOR)); err != nil {
		return err
	}
	if err := e.sink.Put(bitsModels[1], byte(bitsXOR>>8)); err != nil {
		return err
	}
	if err := e.sink.Put(frameModel, frameDelta); err != nil {
		return err
	}
	if err := e.sink.Put(frequency.ModelPlayerCoordSize, byte(e.coordSize)); err != nil {
		return err
	}
	e.player.bits = e.bits
	e.player.frame = e.raw[4]
	e.player.bitsResidual = bitsXOR
	e.player.frameResidual = frameDelta
	return nil
}

func (d *rangePlayerInfoDecoder) decodeHeader() error {
	indexDelta, err := decodeSymbol(d.decoder, d.models, playerIndexModel(d.state))
	if err != nil {
		return err
	}
	index := indexDelta + d.state.previousPlayerIndex
	if index >= rangePlayerSlots {
		return fmt.Errorf("invalid decoded MVD player index %d", index)
	}
	d.state.previousPlayerIndex = index
	d.state.previousPlayerIndexDelta = indexDelta
	d.player = &d.state.players[index]
	bitsModels := playerBitsModels(d.player)
	frameModel := playerFrameModel(d.player)
	bitsLo, err := decodeSymbol(d.decoder, d.models, bitsModels[0])
	if err != nil {
		return err
	}
	bitsHi, err := decodeSymbol(d.decoder, d.models, bitsModels[1])
	if err != nil {
		return err
	}
	bitsXOR := uint16(bitsLo) | uint16(bitsHi)<<8
	d.bits = d.player.bits ^ bitsXOR
	frameDelta, err := decodeSymbol(d.decoder, d.models, frameModel)
	if err != nil {
		return err
	}
	coordSize, err := decodeSymbol(
		d.decoder, d.models, frequency.ModelPlayerCoordSize,
	)
	if err != nil {
		return err
	}
	if !validCoordSize(int(coordSize)) {
		return fmt.Errorf("invalid decoded MVD coordinate size %d", coordSize)
	}
	d.player.bits = d.bits
	d.player.frame += frameDelta
	d.player.bitsResidual = bitsXOR
	d.player.frameResidual = frameDelta
	d.coordSize = int(coordSize)
	d.out = []byte{
		protocol.SVCPlayerInfo,
		index,
		byte(d.bits),
		byte(d.bits >> 8),
		d.player.frame,
	}
	return nil
}

func (e *rangePlayerInfoEncoder) encodeOrigin(axis int) error {
	models := playerOriginModelsFor(e.player, axis, e.coordSize)
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

func (d *rangePlayerInfoDecoder) decodeOrigin(axis int) error {
	models := playerOriginModelsFor(d.player, axis, d.coordSize)
	symbol, err := decodeMotionWord(
		d.decoder,
		d.models,
		models,
		playerOriginLowHighNonzeroModels[axis],
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

func (e *rangePlayerInfoEncoder) encodeOrigin16(
	axis int,
	models modelPair,
) error {
	player := e.player
	value := uint32(binary.LittleEndian.Uint16(e.raw[e.offset:]))
	delta := uint16(value - player.origin[axis])
	interval := e.state.currentTime - player.originTime[axis]
	prediction := predictMotion16(
		uint16(player.originDelta[axis]), interval, player.originInterval[axis],
	)
	residual := delta - prediction
	symbol := zigzag16(residual)

	if err := putMotionWord(
		e.sink,
		models,
		playerOriginLowHighNonzeroModels[axis],
		symbol,
	); err != nil {
		return err
	}
	player.originResidual[axis] = uint32(residual)
	player.originDelta[axis] = uint32(delta)
	player.originInterval[axis] = interval
	player.originTime[axis] = e.state.currentTime
	player.origin[axis] = value
	return nil
}

func (d *rangePlayerInfoDecoder) restoreOrigin16(axis int, symbol uint16) {
	player := d.player
	residual := unzigzag16(symbol)
	interval := d.state.currentTime - player.originTime[axis]
	prediction := predictMotion16(
		uint16(player.originDelta[axis]), interval, player.originInterval[axis],
	)
	delta := residual + prediction
	value := uint16(player.origin[axis]) + delta

	player.originResidual[axis] = uint32(residual)
	player.originDelta[axis] = uint32(delta)
	player.originInterval[axis] = interval
	player.originTime[axis] = d.state.currentTime
	player.origin[axis] = uint32(value)

	d.out = binary.LittleEndian.AppendUint16(d.out, value)
}

func (e *rangePlayerInfoEncoder) encodeOrigin32(
	axis int,
	models modelPair,
) error {
	// Raw coordinate bits are not distances, so they are not scaled by time.
	player := e.player
	value := binary.LittleEndian.Uint32(e.raw[e.offset:])
	delta := value - player.origin[axis]
	prediction := player.originDelta[axis]
	residual := delta - prediction
	symbol := zigzag32(residual)

	// v1 byte order is 1, 0, 2, 3: the low word uses motion coding, then the
	// upper bytes follow. Byte 1 selects the model for byte 0.
	if err := putMotionWord(
		e.sink,
		models,
		playerOriginLowHighNonzeroModels[axis],
		uint16(symbol),
	); err != nil {
		return err
	}
	if err := e.sink.Put(
		frequency.ModelPlayerOriginByte2Delta, byte(symbol>>16),
	); err != nil {
		return err
	}
	if err := e.sink.Put(
		frequency.ModelPlayerOriginByte3Delta, byte(symbol>>24),
	); err != nil {
		return err
	}
	player.originResidual[axis] = 0
	player.originDelta[axis] = delta
	player.origin[axis] = value
	return nil
}

func (d *rangePlayerInfoDecoder) decodeOrigin32(
	axis int,
	lowSymbol uint16,
) error {
	byte2, err := decodeSymbol(
		d.decoder, d.models, frequency.ModelPlayerOriginByte2Delta,
	)
	if err != nil {
		return err
	}
	byte3, err := decodeSymbol(
		d.decoder, d.models, frequency.ModelPlayerOriginByte3Delta,
	)
	if err != nil {
		return err
	}
	player := d.player
	symbol := uint32(lowSymbol) | uint32(byte2)<<16 | uint32(byte3)<<24
	residual := unzigzag32(symbol)
	prediction := player.originDelta[axis]
	delta := residual + prediction
	value := player.origin[axis] + delta

	player.originResidual[axis] = 0
	player.originDelta[axis] = delta
	player.origin[axis] = value

	d.out = binary.LittleEndian.AppendUint32(d.out, value)
	return nil
}

func (e *rangePlayerInfoEncoder) encodeAngle(axis int) error {
	player := e.player
	models := playerAngleModelsFor(player, axis)
	value := binary.LittleEndian.Uint16(e.raw[e.offset:])
	delta := value - player.angle[axis]
	interval := e.state.currentTime - player.angleTime[axis]
	prediction := predictMotion16(
		player.angleDelta[axis], interval, player.angleInterval[axis],
	)
	residual := delta - prediction
	symbol := zigzag16(residual)

	if err := putMotionWord(
		e.sink,
		models,
		playerAngleLowHighNonzeroModels[axis],
		symbol,
	); err != nil {
		return err
	}
	player.angleResidual[axis] = residual
	player.angleDelta[axis] = delta
	player.angleInterval[axis] = interval
	player.angleTime[axis] = e.state.currentTime
	player.angle[axis] = value
	e.offset += 2
	return nil
}

func (d *rangePlayerInfoDecoder) decodeAngle(axis int) error {
	player := d.player
	models := playerAngleModelsFor(player, axis)
	symbol, err := decodeMotionWord(
		d.decoder,
		d.models,
		models,
		playerAngleLowHighNonzeroModels[axis],
	)
	if err != nil {
		return err
	}
	residual := unzigzag16(symbol)
	interval := d.state.currentTime - player.angleTime[axis]
	prediction := predictMotion16(
		player.angleDelta[axis], interval, player.angleInterval[axis],
	)
	delta := residual + prediction
	value := player.angle[axis] + delta

	player.angleResidual[axis] = residual
	player.angleDelta[axis] = delta
	player.angleInterval[axis] = interval
	player.angleTime[axis] = d.state.currentTime
	player.angle[axis] = value

	d.out = binary.LittleEndian.AppendUint16(d.out, value)
	return nil
}

func (e *rangePlayerInfoEncoder) encodeByteFields() error {
	fields := [...]struct {
		mask  uint16
		model frequency.Model
		value *byte
		xor   bool
	}{
		{protocol.DFModel, frequency.ModelPlayerModelDelta, &e.player.model, false},
		{protocol.DFSkinNum, frequency.ModelPlayerSkinDelta, &e.player.skin, false},
		{protocol.DFEffects, frequency.ModelPlayerEffectsXOR, &e.player.effects, true},
		{
			protocol.DFWeaponFrame,
			frequency.ModelPlayerWeaponFrameDelta,
			&e.player.weaponFrame,
			false,
		},
	}
	for _, field := range fields {
		if e.bits&field.mask == 0 {
			continue
		}
		value := e.raw[e.offset]
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

func (d *rangePlayerInfoDecoder) decodeByteFields() error {
	fields := [...]struct {
		mask  uint16
		model frequency.Model
		value *byte
		xor   bool
	}{
		{protocol.DFModel, frequency.ModelPlayerModelDelta, &d.player.model, false},
		{protocol.DFSkinNum, frequency.ModelPlayerSkinDelta, &d.player.skin, false},
		{protocol.DFEffects, frequency.ModelPlayerEffectsXOR, &d.player.effects, true},
		{
			protocol.DFWeaponFrame,
			frequency.ModelPlayerWeaponFrameDelta,
			&d.player.weaponFrame,
			false,
		},
	}
	for _, field := range fields {
		if d.bits&field.mask == 0 {
			continue
		}
		symbol, err := decodeSymbol(d.decoder, d.models, field.model)
		if err != nil {
			return err
		}
		if field.xor {
			*field.value ^= symbol
		} else {
			*field.value += symbol
		}
		d.out = append(d.out, *field.value)
	}
	return nil
}
