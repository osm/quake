package frequency

import (
	"encoding/binary"
	"fmt"
)

// ModelDataFormatVersion versions the MVZF binary layout,
// independently of the MVZ version and trained frequency set.
const ModelDataFormatVersion = 1

// MVZF stores MVZ frequency tables, not compressed demos.
// Header with integers in little endian order:
//
//	[magic:4][version:u16][header size:u16][models:u32][symbols:u32]
//
// The header is followed by ModelCount rows of Symbols cumulative uint32 bounds.
const (
	modelDataMagic      = "MVZF"
	modelDataHeaderSize = 16
	modelDataSize       = modelDataHeaderSize + int(ModelCount)*Symbols*4
)

// ParseModelData validates and loads an MVZF artifact.
func ParseModelData(data []byte) (*Models, error) {
	if err := validateModelDataHeader(data); err != nil {
		return nil, err
	}

	models := &Models{}
	data = data[modelDataHeaderSize:]
	for model := Model(0); model < ModelCount; model++ {
		row := &models.Cumulative[model]
		for symbol := range row {
			row[symbol] = binary.LittleEndian.Uint32(data[:4])
			data = data[4:]
		}
		if err := models.validateRow(model); err != nil {
			return nil, err
		}
	}
	return models, nil
}

func validateModelDataHeader(data []byte) error {
	if len(data) != modelDataSize {
		return fmt.Errorf(
			"invalid MVZ model data size %d, want %d", len(data), modelDataSize,
		)
	}
	if string(data[:4]) != modelDataMagic {
		return fmt.Errorf("invalid MVZ model data magic")
	}
	version := binary.LittleEndian.Uint16(data[4:6])
	if version != ModelDataFormatVersion {
		return fmt.Errorf("unsupported MVZ model data version %d", version)
	}
	headerSize := binary.LittleEndian.Uint16(data[6:8])
	if headerSize != modelDataHeaderSize {
		return fmt.Errorf("invalid MVZ model header size %d", headerSize)
	}
	count := binary.LittleEndian.Uint32(data[8:12])
	if count != uint32(ModelCount) {
		return fmt.Errorf("MVZ model count %d, want %d", count, ModelCount)
	}
	symbols := binary.LittleEndian.Uint32(data[12:16])
	if symbols != Symbols {
		return fmt.Errorf("MVZ symbols per model %d, want %d", symbols, Symbols)
	}
	return nil
}
