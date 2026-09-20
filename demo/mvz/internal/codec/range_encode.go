package codec

import (
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
)

func encodeRangeStream(
	input *rangeProfileInput,
	models *frequency.Models,
) ([]byte, error) {
	encoder := rangecoder.NewEncoder()
	sink := &rangeSymbolSink{encoder: encoder, models: models}
	if err := encodeChunkSymbols(sink, input.records, input.trailing); err != nil {
		return nil, err
	}
	return encoder.Finish(), nil
}

func encodeChunkSymbols(
	sink symbolSink,
	records []rangeRecord,
	trailing []byte,
) error {
	state := &rangeCodecState{}
	for i, record := range records {
		if err := encodeRecordSymbols(sink, state, record); err != nil {
			return fmt.Errorf("encode MVD record %d metadata: %w", i, err)
		}
	}
	if err := sink.Put(recordKindModelFor(state), rangeRecordEnd); err != nil {
		return fmt.Errorf("encode MVD range end: %w", err)
	}
	if err := putUvarint(
		sink, frequency.ModelTrailingLength, uint64(len(trailing)),
	); err != nil {
		return fmt.Errorf("encode MVD trailing length: %w", err)
	}
	return nil
}

// Record wire order: kind, timestamp, target, body mode, then body symbols.
// Raw bodies code only a length here; their bytes are in the general literal
// stream. Structured bodies code operations and reconstruct the MVD length.
func encodeRecordSymbols(
	sink symbolSink,
	state *rangeCodecState,
	record rangeRecord,
) error {
	kind := record.command & 0x7
	if kind >= rangeRecordEnd {
		return fmt.Errorf("invalid MVD record kind %d", kind)
	}
	if err := sink.Put(recordKindModelFor(state), kind); err != nil {
		return err
	}
	if err := sink.Put(timestampModel(kind, state), record.timestamp); err != nil {
		return err
	}
	// Timestamp context uses the old record; body prediction uses the new clock.
	state.updateRecordHistory(kind, record.timestamp)
	target := record.command >> 3
	state.currentCommandTarget = target
	targetModel := commandTargetModel(kind)
	if err := sink.Put(targetModel, target); err != nil {
		return err
	}
	bodyMode := byte(rangeBodyRaw)
	if record.structured {
		bodyMode = rangeBodyStructured
	}
	if err := sink.Put(recordBodyModeModel(kind), bodyMode); err != nil {
		return err
	}
	if !record.structured {
		return putUvarint(
			sink,
			recordLengthModel(kind),
			uint64(len(record.payload)),
		)
	}
	encoder := rangeOperationEncoder{
		sink:          sink,
		state:         state,
		nextKindModel: frequency.ModelOperationKind,
	}
	return encoder.encode(record.operations)
}

type rangeOperationEncoder struct {
	sink          symbolSink
	state         *rangeCodecState
	nextKindModel frequency.Model
}

func (encoder *rangeOperationEncoder) encode(operations []rangeOperation) error {
	for index := 0; index < len(operations); {
		next, err := encoder.encodeAt(operations, index)
		if err != nil {
			return err
		}
		index = next
	}
	return encoder.putKind(rangeOperationEnd)
}

func (encoder *rangeOperationEncoder) encodeAt(
	operations []rangeOperation,
	index int,
) (int, error) {
	operation := operations[index]
	if operation.player != nil {
		return encoder.encodePlayerRun(operations, index)
	}
	if operation.packet != nil {
		return index + 1, encoder.encodePacketEntities(*operation.packet)
	}
	if operation.sound != nil {
		return index + 1, encoder.encodeSound(*operation.sound)
	}
	if operation.temp != nil {
		return index + 1, encoder.encodeTempEntity(*operation.temp)
	}
	if operation.fixed != nil {
		return index + 1, encoder.encodeFixed(*operation.fixed)
	}
	if operation.text != nil {
		return index + 1, encoder.putKind(operation.text.opcode)
	}
	if operation.damage != nil {
		return index + 1, encoder.encodeDamage(*operation.damage)
	}
	return index + 1, encoder.encodeRawLength(operation.raw)
}

func (encoder *rangeOperationEncoder) putKind(operation byte) error {
	if err := encoder.sink.Put(encoder.nextKindModel, operation); err != nil {
		return err
	}
	encoder.nextKindModel = operationModelAfter(operation)
	return nil
}

func (encoder *rangeOperationEncoder) encodePlayerRun(
	operations []rangeOperation,
	start int,
) (int, error) {
	end := start + 1
	for end < len(operations) && operations[end].player != nil &&
		end-start < maxPlayerRunLength {
		end++
	}
	if err := encoder.putKind(rangeOperationPlayerInfo); err != nil {
		return start, err
	}
	if err := encoder.putRunLength(byte(end - start)); err != nil {
		return start, err
	}
	// Index prediction restarts for each run; the players' field history persists.
	encoder.state.previousPlayerIndex = 0
	encoder.state.previousPlayerIndexDelta = 0
	for _, operation := range operations[start:end] {
		if err := encodePlayerInfo(
			encoder.sink,
			encoder.state,
			*operation.player,
		); err != nil {
			return start, err
		}
	}
	return end, nil
}

func (encoder *rangeOperationEncoder) putRunLength(length byte) error {
	symbol := zigzag8(length - encoder.state.previousPlayerRunLength)
	if err := encoder.sink.Put(frequency.ModelPlayerRunLengthDelta, symbol); err != nil {
		return err
	}
	encoder.state.previousPlayerRunLength = length
	return nil
}

func (encoder *rangeOperationEncoder) encodePacketEntities(
	packet rangePacketEntities,
) error {
	kind := byte(rangeOperationPacketEntities)
	if packet.delta {
		kind = rangeOperationDeltaEntities
	}
	if err := encoder.putKind(kind); err != nil {
		return err
	}
	return encodePacketEntities(encoder.sink, encoder.state, packet)
}

func (encoder *rangeOperationEncoder) encodeSound(sound rangeSound) error {
	if err := encoder.putKind(rangeOperationSound); err != nil {
		return err
	}
	return encodeSound(encoder.sink, encoder.state, sound)
}

func (encoder *rangeOperationEncoder) encodeTempEntity(temp rangeTempEntity) error {
	if err := encoder.putKind(rangeOperationTempEntity); err != nil {
		return err
	}
	return encodeTempEntity(encoder.sink, encoder.state, temp)
}

func (encoder *rangeOperationEncoder) encodeFixed(
	fixed rangeFixedOperation,
) error {
	if err := encoder.putKind(fixed.opcode); err != nil {
		return err
	}
	return encodeFixedOperation(encoder.sink, encoder.state, fixed)
}

func (encoder *rangeOperationEncoder) encodeDamage(damage rangeDamage) error {
	if err := encoder.putKind(protocol.SVCDamage); err != nil {
		return err
	}
	return encodeDamage(encoder.sink, encoder.state, damage)
}

func (encoder *rangeOperationEncoder) encodeRawLength(raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty raw MVD operation")
	}
	if err := encoder.putKind(rangeOperationRaw); err != nil {
		return err
	}
	return putUvarint(encoder.sink, frequency.ModelRawByte, uint64(len(raw)))
}
