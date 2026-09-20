package codec

import (
	"bytes"

	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/protocol"
)

const rangeTextStreamCount = 3

const (
	rangeTextStreamPrint = iota
	rangeTextStreamStuffText
	rangeTextStreamCenterPrint
)

type rangeTextOperation struct {
	opcode  byte
	payload []byte
}

func isTextOpcode(opcode byte) bool {
	return textStreamIndex(opcode) >= 0
}

func operationModelAfterText(opcode byte) (frequency.Model, bool) {
	switch opcode {
	case protocol.SVCPrint:
		return frequency.ModelOperationKindAfterPrint, true
	case protocol.SVCStuffText:
		return frequency.ModelOperationKindAfterStuffText, true
	case protocol.SVCCenterPrint:
		return frequency.ModelOperationKindAfterCenterPrint, true
	default:
		return 0, false
	}
}

func textStreamIndex(opcode byte) int {
	switch opcode {
	case protocol.SVCPrint:
		return rangeTextStreamPrint
	case protocol.SVCStuffText:
		return rangeTextStreamStuffText
	case protocol.SVCCenterPrint:
		return rangeTextStreamCenterPrint
	default:
		return -1
	}
}

func newTextOperation(raw []byte) *rangeTextOperation {
	if len(raw) < 2 || !isTextOpcode(raw[0]) || raw[len(raw)-1] != 0 {
		return nil
	}
	if raw[0] == protocol.SVCPrint && len(raw) < 3 {
		return nil
	}
	return &rangeTextOperation{opcode: raw[0], payload: raw[1:]}
}

// Text has no range-coded length: its original NUL ends the payload. Print
// also carries a level byte before the string; only the opcode was omitted.
func restoreText(stream []byte, offset *int, opcode byte) ([]byte, bool) {
	if *offset < 0 || *offset >= len(stream) {
		return nil, false
	}
	start := *offset
	stringStart := start
	if opcode == protocol.SVCPrint {
		stringStart++
		if stringStart >= len(stream) {
			return nil, false
		}
	}
	nul := bytes.IndexByte(stream[stringStart:], 0)
	if nul < 0 {
		return nil, false
	}
	end := stringStart + nul + 1
	*offset = end
	out := make([]byte, 1, 1+end-start)
	out[0] = opcode
	out = append(out, stream[start:end]...)
	return out, true
}
