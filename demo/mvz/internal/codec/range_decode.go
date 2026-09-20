package codec

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/rangecoder"
	"github.com/osm/quake/protocol"
	mvdprotocol "github.com/osm/quake/protocol/mvd"
)

type rangeChunkDecoder struct {
	decoder       *rangecoder.Decoder
	models        *frequency.Models
	literals      []byte
	literalOffset int
	textStreams   [rangeTextStreamCount][]byte
	textOffsets   [rangeTextStreamCount]int
	decodedSize   int
	state         rangeCodecState
	out           []byte
}

func (chunk *rangeChunkDecoder) decodeRecords() error {
	for {
		kind, err := decodeSymbol(
			chunk.decoder,
			chunk.models,
			recordKindModelFor(&chunk.state),
		)
		if err != nil {
			return fmt.Errorf("decode MVD record kind: %w", err)
		}
		if kind == rangeRecordEnd {
			return nil
		}
		if kind > rangeRecordEnd {
			return fmt.Errorf("invalid decoded MVD record kind %d", kind)
		}
		if err := chunk.decodeRecord(kind); err != nil {
			return err
		}
	}
}

func (chunk *rangeChunkDecoder) decodeRecord(kind byte) error {
	timestamp, err := decodeSymbol(
		chunk.decoder,
		chunk.models,
		timestampModel(kind, &chunk.state),
	)
	if err != nil {
		return fmt.Errorf("decode MVD timestamp: %w", err)
	}
	// Timestamp context uses the old record; body prediction uses the new clock.
	chunk.state.updateRecordHistory(kind, timestamp)

	targetModel := commandTargetModel(kind)
	target, err := decodeSymbol(chunk.decoder, chunk.models, targetModel)
	if err != nil {
		return fmt.Errorf("decode MVD command target: %w", err)
	}
	if target > 31 {
		return fmt.Errorf("invalid decoded MVD command target %d", target)
	}
	chunk.state.currentCommandTarget = target

	bodyMode, err := chunk.decodeBodyMode(kind)
	if err != nil {
		return err
	}
	if err := chunk.appendOutput([]byte{timestamp, byte(target<<3) | kind}); err != nil {
		return err
	}
	switch bodyMode {
	case rangeBodyRaw:
		return chunk.decodeRawRecord(kind)
	case rangeBodyStructured:
		return chunk.decodeStructuredRecord(kind)
	default:
		return fmt.Errorf("unknown MVZ record body mode %d", bodyMode)
	}
}

func (chunk *rangeChunkDecoder) decodeBodyMode(kind byte) (byte, error) {
	model := recordBodyModeModel(kind)
	bodyMode, err := decodeSymbol(chunk.decoder, chunk.models, model)
	if err != nil {
		return 0, fmt.Errorf("decode MVD record body mode: %w", err)
	}
	return bodyMode, nil
}

func (chunk *rangeChunkDecoder) decodeRawRecord(kind byte) error {
	model := recordLengthModel(kind)
	length, err := decodeUvarint(chunk.decoder, chunk.models, model)
	if err != nil {
		return fmt.Errorf("decode MVD record length: %w", err)
	}
	if length > uint64(chunk.decodedSize-len(chunk.out)) {
		return fmt.Errorf("decoded MVD record length %d exceeds chunk", length)
	}
	data, err := chunk.takeLiteral(length, "raw MVD record")
	if err != nil {
		return err
	}
	return chunk.appendOutput(data)
}

func (chunk *rangeChunkDecoder) decodeStructuredRecord(kind byte) error {
	prefixLength := 0
	if kind == mvdprotocol.DemoMultiple {
		prefixLength = 4
	}
	prefix, err := chunk.takeLiteral(uint64(prefixLength), "MVD record prefix")
	if err != nil {
		return err
	}
	if err := chunk.appendOutput(prefix); err != nil {
		return err
	}
	// The structured grammar omits the four bytes that store the original body's length.
	// Reserve its place, then fill it from the reconstructed operation bytes.
	sizeOffset := len(chunk.out)
	if err := chunk.appendOutput([]byte{0, 0, 0, 0}); err != nil {
		return err
	}
	bodyStart := len(chunk.out)

	if err := chunk.decodeOperations(); err != nil {
		return err
	}
	bodySize := uint32(len(chunk.out) - bodyStart)
	binary.LittleEndian.PutUint32(chunk.out[sizeOffset:sizeOffset+4], bodySize)
	return nil
}

func (chunk *rangeChunkDecoder) decodeOperations() error {
	// The first kind has no predecessor. Later kinds use the preceding operation
	// as context, including when that operation was a run of player updates.
	nextKindModel := frequency.ModelOperationKind
	for {
		operation, err := decodeSymbol(chunk.decoder, chunk.models, nextKindModel)
		if err != nil {
			return fmt.Errorf("decode MVD operation kind: %w", err)
		}
		nextKindModel = operationModelAfter(operation)
		if operation == rangeOperationEnd {
			return nil
		}
		if err := chunk.decodeOperation(operation); err != nil {
			return err
		}
	}
}

func (chunk *rangeChunkDecoder) decodeOperation(operation byte) error {
	switch operation {
	case rangeOperationRaw:
		return chunk.decodeRaw()
	case rangeOperationPlayerInfo:
		return chunk.decodePlayerRun()
	case rangeOperationPacketEntities, rangeOperationDeltaEntities:
		return chunk.decodePacketEntities(operation == rangeOperationDeltaEntities)
	case rangeOperationSound:
		return chunk.decodeSound()
	case rangeOperationTempEntity:
		return chunk.decodeTempEntity()
	case protocol.SVCDamage:
		return chunk.decodeDamage()
	default:
		return chunk.decodeTextOrFixed(operation)
	}
}

func (chunk *rangeChunkDecoder) decodeRaw() error {
	length, err := decodeUvarint(
		chunk.decoder, chunk.models, frequency.ModelRawByte,
	)
	if err != nil {
		return fmt.Errorf("decode raw MVD operation length: %w", err)
	}
	if length == 0 {
		return fmt.Errorf("decoded empty raw MVD operation")
	}
	data, err := chunk.takeLiteral(length, "raw MVD operation")
	if err != nil {
		return err
	}
	return chunk.appendOutput(data)
}

func (chunk *rangeChunkDecoder) decodePlayerRun() error {
	count, err := chunk.decodeRunLength()
	if err != nil {
		return err
	}
	for range count {
		data, err := decodePlayerInfo(
			chunk.decoder,
			chunk.models,
			&chunk.state,
		)
		if err != nil {
			return fmt.Errorf("decode MVD svc_playerinfo: %w", err)
		}
		if err := chunk.appendOutput(data); err != nil {
			return err
		}
	}
	return nil
}

func (chunk *rangeChunkDecoder) decodeRunLength() (uint64, error) {
	symbol, err := decodeSymbol(
		chunk.decoder,
		chunk.models,
		frequency.ModelPlayerRunLengthDelta,
	)
	if err != nil {
		return 0, fmt.Errorf("decode MVD player run length delta: %w", err)
	}
	decodedCount := chunk.state.previousPlayerRunLength + unzigzag8(symbol)
	count := uint64(decodedCount)
	if count == 0 || count > uint64(chunk.decodedSize/5+1) {
		return 0, fmt.Errorf("invalid MVD player run length %d", count)
	}
	chunk.state.previousPlayerRunLength = decodedCount
	chunk.state.previousPlayerIndex = 0
	chunk.state.previousPlayerIndexDelta = 0
	return count, nil
}

func (chunk *rangeChunkDecoder) decodePacketEntities(delta bool) error {
	decodedLimit := chunk.decodedSize - len(chunk.out)
	data, err := decodePacketEntities(
		chunk.decoder,
		chunk.models,
		&chunk.state,
		delta,
		decodedLimit,
	)
	if err != nil {
		return fmt.Errorf("decode MVD packet entities: %w", err)
	}
	return chunk.appendOutput(data)
}

func (chunk *rangeChunkDecoder) decodeSound() error {
	data, err := decodeSound(chunk.decoder, chunk.models, &chunk.state)
	if err != nil {
		return fmt.Errorf("decode svc_sound: %w", err)
	}
	return chunk.appendOutput(data)
}

func (chunk *rangeChunkDecoder) decodeTempEntity() error {
	data, err := decodeTempEntity(chunk.decoder, chunk.models, &chunk.state)
	if err != nil {
		return fmt.Errorf("decode svc_tempentity: %w", err)
	}
	return chunk.appendOutput(data)
}

func (chunk *rangeChunkDecoder) decodeDamage() error {
	data, err := decodeDamage(chunk.decoder, chunk.models, &chunk.state)
	if err != nil {
		return fmt.Errorf("decode svc_damage: %w", err)
	}
	return chunk.appendOutput(data)
}

func (chunk *rangeChunkDecoder) decodeTextOrFixed(
	operation byte,
) error {
	if isTextOpcode(operation) {
		return chunk.decodeText(operation)
	}
	if _, ok := fixedPayloadSize(operation); !ok {
		return fmt.Errorf("unknown MVZ operation kind %d", operation)
	}
	data, err := decodeFixedOperation(
		chunk.decoder,
		chunk.models,
		&chunk.state,
		operation,
	)
	if err != nil {
		return fmt.Errorf("decode fixed MVD operation %d: %w", operation, err)
	}
	return chunk.appendOutput(data)
}

func (chunk *rangeChunkDecoder) decodeText(operation byte) error {
	stream := textStreamIndex(operation)
	data, ok := restoreText(
		chunk.textStreams[stream],
		&chunk.textOffsets[stream],
		operation,
	)
	if !ok {
		return fmt.Errorf("invalid text operation %d", operation)
	}
	return chunk.appendOutput(data)
}

func (chunk *rangeChunkDecoder) takeLiteral(
	length uint64,
	label string,
) ([]byte, error) {
	if length > uint64(len(chunk.literals)-chunk.literalOffset) {
		return nil, fmt.Errorf("%s length %d exceeds literal data", label, length)
	}
	end := chunk.literalOffset + int(length)
	data := chunk.literals[chunk.literalOffset:end]
	chunk.literalOffset = end
	return data, nil
}

// Every record and operation except the end marker writes at least one byte.
// Checking each append bounds memory and guarantees progress without separate counters.
func (chunk *rangeChunkDecoder) appendOutput(data []byte) error {
	if len(data) > chunk.decodedSize-len(chunk.out) {
		return fmt.Errorf("decoded MVD data exceeds chunk size %d", chunk.decodedSize)
	}
	chunk.out = append(chunk.out, data...)
	return nil
}

// Literal streams and reconstructed output must be consumed exactly. The
// arithmetic prefix follows different rules: the range coder can supply
// virtual tail zeros, so the number of bytes it consumes is not checked here.
func (chunk *rangeChunkDecoder) finish() error {
	length, err := decodeUvarint(
		chunk.decoder, chunk.models, frequency.ModelTrailingLength,
	)
	if err != nil {
		return fmt.Errorf("decode MVD trailing length: %w", err)
	}
	remaining := len(chunk.literals) - chunk.literalOffset
	if length != uint64(remaining) {
		return fmt.Errorf(
			"decoded MVD trailing length %d, literal data has %d",
			length,
			remaining,
		)
	}
	for stream := range chunk.textStreams {
		if chunk.textOffsets[stream] != len(chunk.textStreams[stream]) {
			return fmt.Errorf(
				"text stream %d consumed %d bytes, want %d",
				stream,
				chunk.textOffsets[stream],
				len(chunk.textStreams[stream]),
			)
		}
	}
	if err := chunk.appendOutput(chunk.literals[chunk.literalOffset:]); err != nil {
		return err
	}
	if len(chunk.out) != chunk.decodedSize {
		return fmt.Errorf(
			"MVZ range decoded size %d, want %d",
			len(chunk.out),
			chunk.decodedSize,
		)
	}
	return nil
}
