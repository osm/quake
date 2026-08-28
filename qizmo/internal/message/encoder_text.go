package message

import (
	"bytes"
	"fmt"
	"math"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangeenc"
)

const (
	centerPrintDictionaryLimit = 0x125
	chatDictionaryLimit        = 0x1fe
	stuffTextDictionaryLimit   = 0x19a
)

func (e *Encoder) encodeSVCPrint(enc *rangeenc.Encoder, data []byte) error {
	if err := enc.EncodeFreqByte(e.ft, freq.SVCPrintMode, data[0]); err != nil {
		return err
	}
	row := uint32(freq.SVCPrintString)
	dictionary := e.state.PrintStrings
	firstSymbol := uint16(printDictionaryStart)
	if data[0] == protocol.PrintChat {
		row = freq.SVCPrintChatString
		dictionary = e.state.PrintChatStrings
		firstSymbol = chatDictionaryStart
	}
	return e.encodeDictionaryString(enc, data[1:], row, dictionary, firstSymbol)
}

func (e *Encoder) encodeSVCStuffText(enc *rangeenc.Encoder, data []byte) error {
	return e.encodeDictionaryString(enc, data, freq.SVCStuffText,
		e.state.StuffTextStrings, freq.Symbols)
}

func (e *Encoder) encodeSVCCenterPrint(enc *rangeenc.Encoder, data []byte) error {
	return e.encodeDictionaryString(enc, data, freq.SVCCenterPrintString,
		e.state.CenterPrintStrings, freq.Symbols)
}

func (e *Encoder) encodeSVCSetInfo(enc *rangeenc.Encoder, data []byte) error {
	if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerIndex, data[0]); err != nil {
		return err
	}
	keyEnd, ok := endCString(data, setInfoKeyOffset)
	if !ok {
		return fmt.Errorf("setinfo key is not terminated")
	}
	if err := e.encodeDictionaryString(enc, data[setInfoKeyOffset:keyEnd], freq.SVCSetInfoKey,
		e.state.SetInfoStrings, freq.Symbols); err != nil {
		return err
	}
	return e.encodeRepeatedRow(enc, data[keyEnd:], freq.SVCSetInfoValue)
}

func (e *Encoder) encodeSVCLightStyle(enc *rangeenc.Encoder, data []byte) error {
	if err := enc.EncodeFreqByte(e.ft, freq.ByteValue, data[0]); err != nil {
		return err
	}
	return e.encodeRepeatedRow(enc, data[1:], freq.ByteValue)
}

func (e *Encoder) encodeSVCUpdateUserInfo(enc *rangeenc.Encoder, data []byte) error {
	if err := enc.EncodeFreqByte(e.ft, freq.SVCPlayerIndex, data[0]); err != nil {
		return err
	}
	if err := e.encodeRepeatedRow(enc,
		data[updateUserInfoUserIDOffset:updateUserInfoStringOffset],
		freq.SVCUpdateUserInfoUserID); err != nil {
		return err
	}
	return e.encodeRepeatedRow(enc, data[updateUserInfoStringOffset:], freq.SVCUpdateUserInfoString)
}

func (e *Encoder) encodeSVCServerInfo(enc *rangeenc.Encoder, data []byte) error {
	return e.encodeRepeatedRow(enc, data, freq.SVCServerInfoString)
}

func (e *Encoder) encodeDictionaryString(
	enc *rangeenc.Encoder,
	data []byte,
	row uint32,
	dictionary map[uint16][]byte,
	firstSymbol uint16,
) error {
	symbols := e.planDictionaryString(data, row, dictionary, firstSymbol)
	for _, symbol := range symbols {
		if err := enc.EncodeFreqSymbol(e.ft, row, uint32(symbol), freq.PairedSymbols); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) planDictionaryString(
	data []byte,
	row uint32,
	dictionary map[uint16][]byte,
	firstSymbol uint16,
) []uint16 {
	switch row {
	case freq.SVCCenterPrintString:
		// Qizmo's static centerprint dictionary contains only symbols
		// 0x100..0x124. Later embedded strings are decoder-only expansions.
		return planQizmoPrefixDictionaryString(data, dictionary, firstSymbol, centerPrintDictionaryLimit)
	case freq.SVCPrintChatString:
		// Qizmo's chat lookup table contains symbols 0x140..0x1fd.
		return planQizmoDictionaryString(data, dictionary, firstSymbol, chatDictionaryLimit)
	case freq.SVCStuffText:
		// The compressor's stufftext lookup table has 154 entries covering
		// symbols 0x100..0x199. Later decoder expansions are not searched.
		return planQizmoDictionaryString(data, dictionary, firstSymbol, stuffTextDictionaryLimit)
	default:
		return e.planOptimalDictionaryString(data, row, dictionary, firstSymbol)
	}
}

func planQizmoDictionaryString(
	data []byte,
	dictionary map[uint16][]byte,
	firstSymbol, symbolLimit uint16,
) []uint16 {
	entries := buildQizmoDictionary(dictionary, firstSymbol, symbolLimit)
	symbols := make([]uint16, 0, len(data))
	for offset := 0; offset < len(data); {
		if entry, ok := qizmoDictionaryLookup(entries, data[offset:]); ok {
			symbols = append(symbols, entry.symbol)
			offset += len(entry.expansion)
			continue
		}
		symbols = append(symbols, uint16(data[offset]))
		offset++
	}
	return symbols
}

func planQizmoPrefixDictionaryString(
	data []byte,
	dictionary map[uint16][]byte,
	firstSymbol, symbolLimit uint16,
) []uint16 {
	entries := buildQizmoDictionary(dictionary, firstSymbol, symbolLimit)
	symbols := make([]uint16, 0, len(data))
	offset := 0
	if entry, ok := qizmoDictionaryLookup(entries, data); ok {
		symbols = append(symbols, entry.symbol)
		offset = len(entry.expansion)
	}
	for ; offset < len(data); offset++ {
		symbols = append(symbols, uint16(data[offset]))
	}
	return symbols
}

func (e *Encoder) planOptimalDictionaryString(
	data []byte,
	row uint32,
	dictionary map[uint16][]byte,
	firstSymbol uint16,
) []uint16 {
	rowIndex := freq.RowIndex(row)
	cumulative0 := &e.ft.Cumulative[rowIndex]
	cumulative1 := &e.ft.Cumulative[rowIndex+1]
	cost := make([]float64, len(data)+1)
	choice := make([]uint16, len(data))
	advance := make([]int, len(data))
	for i := len(data) - 1; i >= 0; i-- {
		choice[i] = uint16(data[i])
		advance[i] = 1
		cost[i] = pairedSymbolCost(cumulative0, cumulative1, choice[i]) + cost[i+1]
		for symbolValue := int(firstSymbol); symbolValue < freq.PairedSymbols; symbolValue++ {
			symbol := uint16(symbolValue)
			expansion := dictionary[symbol]
			if len(expansion) == 0 || len(expansion) > len(data)-i ||
				!bytes.Equal(data[i:i+len(expansion)], expansion) {
				continue
			}
			candidateCost := pairedSymbolCost(cumulative0, cumulative1, symbol) + cost[i+len(expansion)]
			if candidateCost < cost[i] {
				cost[i] = candidateCost
				choice[i] = symbol
				advance[i] = len(expansion)
			}
		}
	}
	symbols := make([]uint16, 0, len(data))
	for i := 0; i < len(data); i += advance[i] {
		symbols = append(symbols, choice[i])
	}
	return symbols
}

func pairedSymbolCost(
	cumulative0, cumulative1 *[freq.Symbols]uint32,
	symbol uint16,
) float64 {
	var upper uint32
	if symbol < freq.Symbols {
		upper = cumulative0[symbol]
	} else {
		upper = cumulative1[symbol-freq.Symbols]
	}
	var lower uint32
	if symbol != 0 {
		if symbol <= freq.Symbols {
			lower = cumulative0[symbol-1]
		} else {
			lower = cumulative1[symbol-freq.Symbols-1]
		}
	}
	return 32 - math.Log2(float64(uint64(upper)-uint64(lower)))
}

type qizmoDictionaryEntry struct {
	symbol    uint16
	expansion []byte
}

func buildQizmoDictionary(
	dictionary map[uint16][]byte,
	firstSymbol uint16,
	symbolLimit uint16,
) []qizmoDictionaryEntry {
	entries := make([]qizmoDictionaryEntry, 0, int(symbolLimit-firstSymbol))
	for symbol := firstSymbol; symbol < symbolLimit; symbol++ {
		expansion := dictionary[symbol]
		if len(expansion) == 0 {
			continue
		}
		entry := qizmoDictionaryEntry{symbol: symbol, expansion: expansion}
		low, high := 0, len(entries)
		for low != high {
			middle := (low + high) / 2
			comparison := bytes.Compare(entries[middle].expansion, expansion)
			if comparison == 0 {
				low = middle
				break
			}
			if comparison < 0 {
				low = middle + 1
			} else {
				high = middle
			}
		}
		entries = append(entries, qizmoDictionaryEntry{})
		copy(entries[low+1:], entries[low:])
		entries[low] = entry
	}
	return entries
}

func qizmoDictionaryLookup(
	entries []qizmoDictionaryEntry,
	input []byte,
) (qizmoDictionaryEntry, bool) {
	low, high := 0, len(entries)
	for low != high {
		middle := (low + high) / 2
		comparison := compareQizmoDictionaryPrefix(entries[middle].expansion, input)
		if comparison < 0 {
			low = middle + 1
			continue
		}
		if comparison > 0 {
			high = middle
			continue
		}
		best := entries[middle]
		for next := middle + 1; next < high; next++ {
			candidate := entries[next]
			if len(candidate.expansion) <= len(best.expansion) ||
				compareQizmoDictionaryPrefix(candidate.expansion, input) != 0 {
				break
			}
			best = candidate
		}
		return best, true
	}
	return qizmoDictionaryEntry{}, false
}

func compareQizmoDictionaryPrefix(candidate, input []byte) int {
	for i, candidateByte := range candidate {
		var inputByte byte
		if i < len(input) {
			inputByte = input[i]
		}
		if candidateByte < inputByte {
			return -1
		}
		if candidateByte > inputByte {
			return 1
		}
	}
	return 0
}
