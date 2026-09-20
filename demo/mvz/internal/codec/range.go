package codec

import (
	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/protocol"
	mvdprotocol "github.com/osm/quake/protocol/mvd"
)

const (
	rangeRecordEnd               = 7
	rangeBodyRaw                 = 0
	rangeBodyStructured          = 1
	rangeOperationEnd            = 0
	rangeOperationRaw            = 1
	rangeOperationSound          = protocol.SVCSound
	rangeOperationTempEntity     = protocol.SVCTempEntity
	rangeOperationPlayerInfo     = protocol.SVCPlayerInfo
	rangeOperationPacketEntities = protocol.SVCPacketEntities
	rangeOperationDeltaEntities  = protocol.SVCDeltaPacketEntities
	maxPlayerRunLength           = 1<<8 - 1
)

var recordKindContextModels = [rangeRecordEnd]frequency.Model{
	frequency.ModelRecordKindAfter0,
	frequency.ModelRecordKindAfter1,
	frequency.ModelRecordKindAfter2,
	frequency.ModelRecordKindAfter3,
	frequency.ModelRecordKindAfter4,
	frequency.ModelRecordKindAfter5,
	frequency.ModelRecordKindAfter6,
}

var recordBodyModeContextModels = [rangeRecordEnd]frequency.Model{
	frequency.ModelRecordBodyMode,
	frequency.ModelRecordBodyModeKind1,
	frequency.ModelRecordBodyModeKind2,
	frequency.ModelRecordBodyModeKind3,
	frequency.ModelRecordBodyModeKind4,
	frequency.ModelRecordBodyModeKind5,
	frequency.ModelRecordBodyModeKind6,
}

var timestampContextModels = [rangeRecordEnd]frequency.Model{
	frequency.ModelTimestamp,
	frequency.ModelTimestampKind1,
	frequency.ModelTimestampKind2,
	frequency.ModelTimestampKind3,
	frequency.ModelTimestampKind4,
	frequency.ModelTimestampKind5,
	frequency.ModelTimestampKind6,
}

var timestampAfterNonzeroModels = [rangeRecordEnd]frequency.Model{
	frequency.ModelTimestampKind0PrevNonzero,
	frequency.ModelTimestampKind1PrevNonzero,
	frequency.ModelTimestampKind2PrevNonzero,
	frequency.ModelTimestampKind3PrevNonzero,
	frequency.ModelTimestampKind4PrevNonzero,
	frequency.ModelTimestampKind5PrevNonzero,
	frequency.ModelTimestampKind6PrevNonzero,
}

var timestampSinglePreviousKindModels = [rangeRecordEnd]frequency.Model{
	frequency.ModelTimestampKind4PrevKind0,
	frequency.ModelTimestampKind4PrevKind1,
	frequency.ModelTimestampKind4PrevKind2,
	frequency.ModelTimestampKind4PrevKind3,
	frequency.ModelTimestampKind4PrevKind4,
	frequency.ModelTimestampKind4PrevKind5,
	frequency.ModelTimestampKind4PrevKind6,
}

var timestampAllPreviousKindModels = [rangeRecordEnd]frequency.Model{
	frequency.ModelTimestampKind6PrevKind0,
	frequency.ModelTimestampKind6PrevKind1,
	frequency.ModelTimestampKind6PrevKind2,
	frequency.ModelTimestampKind6PrevKind3,
	frequency.ModelTimestampKind6PrevKind4,
	frequency.ModelTimestampKind6PrevKind5,
	frequency.ModelTimestampKind6PrevKind6,
}

var commandTargetContextModels = [rangeRecordEnd]frequency.Model{
	frequency.ModelCommandTarget,
	frequency.ModelCommandTargetKind1,
	frequency.ModelCommandTargetKind2,
	frequency.ModelCommandTargetKind3,
	frequency.ModelCommandTargetKind4,
	frequency.ModelCommandTargetKind5,
	frequency.ModelCommandTargetKind6,
}

func recordKindModelFor(state *rangeCodecState) frequency.Model {
	if !state.haveRecordKind {
		return frequency.ModelRecordKind
	}
	return recordKindContextModels[state.previousRecordKind]
}

func recordBodyModeModel(kind byte) frequency.Model {
	return recordBodyModeContextModels[kind]
}

func timestampModel(kind byte, state *rangeCodecState) frequency.Model {
	if state.previousTimestamp != 0 {
		return timestampAfterNonzeroModels[kind]
	}
	if !state.haveRecordKind {
		return timestampContextModels[kind]
	}
	switch kind {
	case mvdprotocol.DemoSingle:
		return timestampSinglePreviousKindModels[state.previousRecordKind]
	case mvdprotocol.DemoAll:
		return timestampAllPreviousKindModels[state.previousRecordKind]
	default:
		return timestampContextModels[kind]
	}
}

func commandTargetModel(kind byte) frequency.Model {
	return commandTargetContextModels[kind]
}

func operationModelAfter(operation byte) frequency.Model {
	switch operation {
	case rangeOperationRaw:
		return frequency.ModelOperationKindAfterRaw
	case rangeOperationPlayerInfo:
		return frequency.ModelOperationKindAfterPlayer
	case rangeOperationPacketEntities:
		return frequency.ModelOperationKindAfterPacket
	case rangeOperationDeltaEntities:
		return frequency.ModelOperationKindAfterDelta
	case rangeOperationSound:
		return frequency.ModelOperationKindAfterSound
	case rangeOperationTempEntity:
		return frequency.ModelOperationKindAfterTempEntity
	case protocol.SVCDamage:
		return frequency.ModelOperationKindAfterDamage
	}
	if model, ok := operationModelAfterText(operation); ok {
		return model
	}
	if model, ok := operationModelAfterFixed(operation); ok {
		return model
	}
	return frequency.ModelOperationKind
}

func recordLengthModel(kind byte) frequency.Model {
	switch kind {
	case protocol.DemoRead:
		// Historical table names: Set codes demo_read lengths, and Command
		// codes demo_set lengths. These names identify frozen frequency rows;
		// swapping the rows to match the names would change the v1 grammar.
		return frequency.ModelRecordLengthSet
	case protocol.DemoSet:
		return frequency.ModelRecordLengthCommand
	case mvdprotocol.DemoMultiple:
		return frequency.ModelRecordLengthMultiple
	case mvdprotocol.DemoSingle:
		return frequency.ModelRecordLengthSingle
	case mvdprotocol.DemoStats:
		return frequency.ModelRecordLengthStats
	case mvdprotocol.DemoAll:
		return frequency.ModelRecordLengthAll
	default:
		return frequency.ModelLength
	}
}
