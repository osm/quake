package freq

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Tables struct {
	Cumulative [256][Symbols]uint32
}

func buildCumulativeRowFromCounts(counts [Symbols]uint32) [Symbols]uint32 {
	var total uint64
	for _, v := range counts {
		total += uint64(v)
	}
	if total == 0 {
		var row [Symbols]uint32
		for i := range row {
			row[i] = 0xffffffff
		}
		return row
	}

	var row [Symbols]uint32
	var accum uint64
	for i, v := range counts {
		accum += uint64(v)
		row[i] = uint32((accum * 0xffffffff) / total)
	}
	return row
}

func loadRawRows(data []byte) (*[256][Symbols]uint32, error) {
	if len(data) != 0x40000 {
		return nil, fmt.Errorf("invalid compress.dat size %d", len(data))
	}

	var raw [256][Symbols]uint32
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &raw); err != nil {
		return nil, fmt.Errorf("read compress.dat: %w", err)
	}
	return &raw, nil
}

func isPairedRow(row int) bool {
	switch row {
	case RowIndex(SVCPrintString),
		RowIndex(SVCPrintString3),
		RowIndex(SVCStufftext),
		RowIndex(SVCCenterPrintString),
		RowIndex(SVCSetInfoKey):
		return true
	default:
		return false
	}
}

func buildPairedCumulativeRows(tables *Tables, raw *[256][Symbols]uint32, row int) {
	var pair [Symbols * 2]uint32
	copy(pair[:Symbols], raw[row][:])
	copy(pair[Symbols:], raw[row+1][:])

	var total uint64
	for _, v := range pair {
		total += uint64(v)
	}

	var accum uint64
	for i, v := range pair {
		accum += uint64(v)
		cum := uint32((accum * 0xffffffff) / total)
		if i < Symbols {
			tables.Cumulative[row][i] = cum
			continue
		}
		tables.Cumulative[row+1][i-Symbols] = cum
	}
}

func buildTwoSymbolCumulativeRow(raw [Symbols]uint32) [Symbols]uint32 {
	row := buildCumulativeRowFromCounts([Symbols]uint32{raw[0], raw[1]})
	row[1] = 0xffffffff
	return row
}

func NewTables(data []byte) (*Tables, error) {
	raw, err := loadRawRows(data)
	if err != nil {
		return nil, err
	}

	raw[RowIndex(SVCPlayerInfoBackref)][1] += 0x100

	tables := &Tables{}
	damageRow := RowIndex(SVCDamageHasFrom)
	for row := 0; row < damageRow; row++ {
		// A small set of string-heavy rows in compress.dat are stored as paired
		// 2x256 tables and must be rebuilt together.
		if isPairedRow(row) {
			buildPairedCumulativeRows(tables, raw, row)
			row++
			continue
		}
		tables.Cumulative[row] = buildCumulativeRowFromCounts(raw[row])
	}

	// svc_damage only uses the first two symbols in its row.
	tables.Cumulative[damageRow] = buildTwoSymbolCumulativeRow(raw[damageRow])

	for row := RowIndex(SVCQizmoVoiceData); row < len(tables.Cumulative); row++ {
		tables.Cumulative[row] = buildCumulativeRowFromCounts(raw[row])
	}

	return tables, nil
}

func RowIndex(freqTableAddr uint32) int {
	d := freqTableAddr - ModelTableBase
	return int(d / RowSize)
}

func (ft *Tables) CumulativeRow(freqTableAddr uint32) []uint32 {
	return ft.Cumulative[RowIndex(freqTableAddr)][:]
}
