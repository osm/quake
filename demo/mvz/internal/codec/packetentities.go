package codec

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
)

const (
	rangeEntitySlots     = protocol.UCheckMoreBits + 1
	maxPacketEntityCount = 1<<16 - 1
)

type rangePacketEntities struct {
	delta    bool
	base     byte
	entities []rangePacketEntity
}

type rangePacketEntityEncoder struct {
	sink                symbolSink
	state               *rangeCodecState
	previousNumber      uint16
	previousNumberDelta uint16
}

type rangePacketEntityDecoder struct {
	decoder             *rangecoder.Decoder
	models              *frequency.Models
	state               *rangeCodecState
	previousNumber      uint16
	previousNumberDelta uint16
}

func encodePacketEntities(
	sink symbolSink,
	state *rangeCodecState,
	packet rangePacketEntities,
) error {
	if len(packet.entities) > maxPacketEntityCount {
		return fmt.Errorf(
			"packet-entity count %d exceeds wire width",
			len(packet.entities),
		)
	}
	if packet.delta {
		baseDelta := packet.base - state.entityDeltaBase
		if err := sink.Put(frequency.ModelEntityDeltaBase, baseDelta); err != nil {
			return err
		}
		state.entityDeltaBase = packet.base
	}
	entityCount := uint16(len(packet.entities))
	countSymbol := zigzag16(entityCount - state.previousEntityCount)
	if err := putUvarint(
		sink, frequency.ModelEntityCountDelta, uint64(countSymbol),
	); err != nil {
		return err
	}
	state.previousEntityCount = entityCount
	entityEncoder := rangePacketEntityEncoder{sink: sink, state: state}
	for i, entity := range packet.entities {
		if err := entityEncoder.encode(entity); err != nil {
			return fmt.Errorf("entity %d: %w", i, err)
		}
	}
	return nil
}

func decodePacketEntities(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	delta bool,
	decodedLimit int,
) ([]byte, error) {
	minimumSize := packetEntitiesMinimumSize(delta)
	if decodedLimit < minimumSize {
		return nil, fmt.Errorf("packet entities exceed decoded MVD chunk size")
	}
	out, err := decodePacketEntitiesPrefix(decoder, models, state, delta)
	if err != nil {
		return nil, err
	}
	maxEntities := uint64((decodedLimit - minimumSize) / 2)
	count, err := decodePacketEntityCount(decoder, models, state, maxEntities)
	if err != nil {
		return nil, err
	}
	entityDecoder := rangePacketEntityDecoder{
		decoder: decoder,
		models:  models,
		state:   state,
	}
	for i := uint64(0); i < count; i++ {
		if len(out) > decodedLimit-2 {
			return nil, fmt.Errorf("packet entities exceed decoded MVD chunk size")
		}
		remaining := decodedLimit - len(out) - 2
		entity, err := entityDecoder.decode(remaining)
		if err != nil {
			return nil, fmt.Errorf("entity %d: %w", i, err)
		}
		out = append(out, entity...)
	}
	return binary.LittleEndian.AppendUint16(out, 0), nil
}

func packetEntitiesMinimumSize(delta bool) int {
	if delta {
		return 4 // opcode, delta sequence byte, and two bytes for the terminator
	}
	return 3 // opcode and two bytes for the terminator
}

func decodePacketEntitiesPrefix(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	delta bool,
) ([]byte, error) {
	operation := byte(protocol.SVCPacketEntities)
	if !delta {
		return []byte{operation}, nil
	}
	base, err := decodeSymbol(decoder, models, frequency.ModelEntityDeltaBase)
	if err != nil {
		return nil, err
	}
	base += state.entityDeltaBase
	state.entityDeltaBase = base
	return []byte{protocol.SVCDeltaPacketEntities, base}, nil
}

func decodePacketEntityCount(
	decoder *rangecoder.Decoder,
	models *frequency.Models,
	state *rangeCodecState,
	maximum uint64,
) (uint64, error) {
	countSymbol, err := decodeUvarint(decoder, models, frequency.ModelEntityCountDelta)
	if err != nil {
		return 0, err
	}
	if countSymbol > maxPacketEntityCount {
		return 0, fmt.Errorf(
			"packet-entity count delta %d exceeds wire width",
			countSymbol,
		)
	}
	previousCount := state.previousEntityCount
	count := uint64(previousCount + unzigzag16(uint16(countSymbol)))
	if count > maximum {
		return 0, fmt.Errorf("packet-entity count %d exceeds chunk size", count)
	}
	state.previousEntityCount = uint16(count)
	return count, nil
}

func (e *rangePacketEntityEncoder) encode(entity rangePacketEntity) error {
	layout, err := entity.layout()
	if err != nil {
		return err
	}
	number := layout.header & protocol.UCheckMoreBits
	if err := e.encodeNumber(number); err != nil {
		return err
	}
	entityState := &e.state.entities[number]
	bits := byte(layout.header >> 9)
	if err := e.sink.Put(
		frequency.ModelEntityBitsXOR, bits^entityState.bits,
	); err != nil {
		return err
	}
	entityState.bits = bits
	if err := encodeEntityFlags(e.sink, entityState, layout); err != nil {
		return err
	}
	if layout.header&protocol.URemove != 0 {
		return nil
	}
	return e.encodeFields(entityState, entity, layout)
}

func (d *rangePacketEntityDecoder) decode(decodedLimit int) ([]byte, error) {
	if decodedLimit < 2 {
		return nil, fmt.Errorf("packet entity exceeds decoded MVD chunk size")
	}
	number, err := d.decodeNumber()
	if err != nil {
		return nil, err
	}
	entityState := &d.state.entities[number]
	header, err := d.decodeHeader(number, entityState)
	if err != nil {
		return nil, err
	}
	out := binary.LittleEndian.AppendUint16(nil, header)
	flags, err := decodeEntityFlags(d.decoder, d.models, entityState, header)
	if err != nil {
		return nil, err
	}
	out = append(out, flags...)
	if len(out) > decodedLimit {
		return nil, fmt.Errorf("packet-entity header exceeds decoded MVD chunk size")
	}
	if header&protocol.URemove != 0 {
		return out, nil
	}
	fields, err := d.decodeFields(
		entityState,
		header,
		flags,
		decodedLimit-len(out),
	)
	if err != nil {
		return nil, fmt.Errorf("fields: %w", err)
	}
	return append(out, fields...), nil
}

func (e *rangePacketEntityEncoder) encodeNumber(number uint16) error {
	delta := number - e.previousNumber
	context := residualMagnitudeContext16(e.previousNumberDelta)
	numberModels := entityNumberContextModels[context]
	if err := putWord(e.sink, numberModels, delta); err != nil {
		return err
	}
	e.previousNumber = number
	e.previousNumberDelta = delta
	return nil
}

func (d *rangePacketEntityDecoder) decodeNumber() (uint16, error) {
	context := residualMagnitudeContext16(d.previousNumberDelta)
	numberModels := entityNumberContextModels[context]
	delta, err := decodeWord(d.decoder, d.models, numberModels)
	if err != nil {
		return 0, err
	}
	number := d.previousNumber + delta
	if number >= rangeEntitySlots {
		return 0, fmt.Errorf("invalid decoded packet-entity number %d", number)
	}
	d.previousNumber = number
	d.previousNumberDelta = delta
	return number, nil
}

func (d *rangePacketEntityDecoder) decodeHeader(
	number uint16,
	entityState *rangeEntityState,
) (uint16, error) {
	bitsXOR, err := decodeSymbol(
		d.decoder, d.models, frequency.ModelEntityBitsXOR,
	)
	if err != nil {
		return 0, err
	}
	bits := bitsXOR ^ entityState.bits
	if bits&0x80 != 0 {
		return 0, fmt.Errorf("invalid decoded packet-entity header bits %#x", bits)
	}
	header := number | uint16(bits)<<9
	if header == 0 {
		return 0, fmt.Errorf("decoded packet-entity collides with terminator")
	}
	entityState.bits = bits
	return header, nil
}

func (e *rangePacketEntityEncoder) encodeFields(
	entityState *rangeEntityState,
	entity rangePacketEntity,
	layout rangeEntityLayout,
) error {
	if err := e.sink.Put(
		frequency.ModelEntityCoordSize, byte(entity.coordSize),
	); err != nil {
		return err
	}
	if err := e.sink.Put(
		frequency.ModelEntityAngleSize, byte(entity.angleSize),
	); err != nil {
		return err
	}
	if err := encodeEntityFields(
		e.sink,
		entityState,
		layout,
		entity.coordSize,
		entity.angleSize,
		e.state.currentTime,
	); err != nil {
		return fmt.Errorf("fields: %w", err)
	}
	return nil
}

func (d *rangePacketEntityDecoder) decodeFields(
	entityState *rangeEntityState,
	header uint16,
	flags []byte,
	decodedLimit int,
) ([]byte, error) {
	coordSize, err := decodeSymbol(
		d.decoder, d.models, frequency.ModelEntityCoordSize,
	)
	if err != nil {
		return nil, err
	}
	if !validCoordSize(int(coordSize)) {
		return nil, fmt.Errorf(
			"invalid decoded packet-entity coordinate size %d", coordSize,
		)
	}
	angleSize, err := decodeSymbol(
		d.decoder, d.models, frequency.ModelEntityAngleSize,
	)
	if err != nil {
		return nil, err
	}
	if !validEntityAngleSize(int(angleSize)) {
		return nil, fmt.Errorf(
			"invalid decoded packet-entity angle size %d", angleSize,
		)
	}
	bits := header &^ protocol.UCheckMoreBits
	if len(flags) != 0 {
		bits |= uint16(flags[0])
	}
	return decodeEntityFields(
		d.decoder,
		d.models,
		entityState,
		bits,
		int(coordSize),
		int(angleSize),
		d.state.currentTime,
		decodedLimit,
	)
}
