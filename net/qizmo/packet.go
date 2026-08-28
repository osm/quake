package qizmo

import (
	"encoding/binary"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
)

const (
	rawServerHeaderSize        = protocol.QWServerPacketHeaderSize
	rawClientHeaderSize        = protocol.QWClientPacketHeaderSize
	qPortSize                  = rawClientHeaderSize - rawServerHeaderSize
	compressedClientHeaderSize = qizmoprotocol.LinkHeaderSize + qPortSize
	clientTMovePayloadSize     = 3 * 2
	clientUploadHeaderSize     = 3
	sequenceMask               = protocol.QWSequenceMask
	sequenceHalfRange          = protocol.QWSequenceReliableBit >> 1
	connectionlessSequence     = protocol.QWConnectionlessSequence
)

type rawPacketHeader struct {
	sequence uint32
	ack      uint32
}

type rawClientHeader struct {
	rawPacketHeader
	qPort uint16
}

func readRawServerHeader(packet []byte) (rawPacketHeader, bool) {
	return readRawPacketHeader(packet, rawServerHeaderSize)
}

func readRawClientHeader(packet []byte) (rawClientHeader, bool) {
	header, ok := readRawPacketHeader(packet, rawClientHeaderSize)
	if !ok {
		return rawClientHeader{}, false
	}
	return rawClientHeader{
		rawPacketHeader: header,
		qPort:           binary.LittleEndian.Uint16(packet[rawServerHeaderSize:]),
	}, true
}

func readRawPacketHeader(packet []byte, size int) (rawPacketHeader, bool) {
	if len(packet) < size {
		return rawPacketHeader{}, false
	}
	header := rawPacketHeader{
		sequence: binary.LittleEndian.Uint32(packet),
		ack:      binary.LittleEndian.Uint32(packet[protocol.QWPacketAckOffset:]),
	}
	return header, header.sequence != connectionlessSequence
}

func encodeCompressedClientHeader(header rawClientHeader, clcDelta bool) []byte {
	link := qizmoprotocol.EncodeHeader(qizmoprotocol.LinkHeader{
		Sequence: header.sequence,
		Ack:      header.ack,
		CLCDelta: clcDelta,
	})
	compressed := make([]byte, 0, compressedClientHeaderSize)
	compressed = append(compressed, link[:]...)
	return binary.LittleEndian.AppendUint16(compressed, header.qPort)
}
