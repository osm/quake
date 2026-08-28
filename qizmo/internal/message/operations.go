package message

import (
	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
)

type fixedOperationCodec struct {
	size      int
	repeatRow uint32
	rows      []uint32
}

func fixedRows(rows ...uint32) fixedOperationCodec {
	return fixedOperationCodec{size: len(rows), rows: rows}
}

func repeatedRow(size int, row uint32) fixedOperationCodec {
	return fixedOperationCodec{size: size, repeatRow: row}
}

func (c fixedOperationCodec) row(index int) uint32 {
	if c.rows != nil {
		return c.rows[index]
	}
	return c.repeatRow
}

var fixedOperationCodecs = map[byte]fixedOperationCodec{
	protocol.SVCNOP:           fixedRows(),
	protocol.SVCDisconnect:    fixedRows(),
	protocol.SVCKilledMonster: fixedRows(),
	protocol.SVCFoundSecret:   fixedRows(),
	protocol.SVCSellScreen:    fixedRows(),
	protocol.SVCSmallKick:     fixedRows(),
	protocol.SVCBigKick:       fixedRows(),

	protocol.SVCUpdateStat:  fixedRows(freq.SVCUpdateStatIndex, freq.SVCStatValue),
	protocol.SVCUpdateFrags: fixedRows(freq.SVCPlayerIndex, freq.SVCFragsValueLo, freq.SVCFragsValueHi),
	protocol.SVCUpdateStatLong: fixedRows(
		freq.SVCUpdateStatLongIndex,
		freq.SVCUpdateStatLongByte0,
		freq.SVCUpdateStatLongByte1,
		freq.SVCUpdateStatLongByte2,
		freq.SVCUpdateStatLongByte3,
	),

	protocol.SVCSpawnStaticSound: repeatedRow(9, freq.ByteValue),
	protocol.SVCIntermission:     repeatedRow(9, freq.ByteValue),
	protocol.SVCMaxSpeed:         repeatedRow(4, freq.ByteValue),
	protocol.SVCEntGravity:       repeatedRow(4, freq.ByteValue),
	protocol.SVCSetAngle:         repeatedRow(3, freq.ByteValue),
	protocol.SVCStopSound:        repeatedRow(2, freq.ByteValue),
	protocol.SVCSetPause:         repeatedRow(1, freq.ByteValue),
	protocol.SVCCDTrack:          repeatedRow(1, freq.ByteValue),
	protocol.SVCChokeCount:       repeatedRow(1, freq.SVCChokeCount),
	protocol.SVCUpdateEnterTime:  repeatedRow(5, freq.ByteValue),
	qizmoprotocol.SVCBlock:       repeatedRow(qizmoprotocol.SVCBlockPayloadSize, freq.ByteValue),
}
