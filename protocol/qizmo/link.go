package qizmo

import (
	"encoding/binary"
	"errors"

	"github.com/osm/quake/protocol"
)

const (
	linkSequenceBits      = 14
	linkSequenceMask      = uint32(1<<linkSequenceBits - 1)
	linkWrapDistance      = uint32(1 << (linkSequenceBits - 1))
	linkWireReliableBit   = uint16(1 << 15)
	linkCLCDeltaBit       = uint16(1 << 14)
	linkCompressionMarker = uint16(1 << 14)
	linkAckOffset         = 2
	LinkHeaderSize        = 4
)

var ErrNotCompressed = errors.New("not a Qizmo-compressed link packet")

type LinkHeader struct {
	Sequence uint32
	Ack      uint32
	CLCDelta bool
}

func EncodeHeader(header LinkHeader) [LinkHeaderSize]byte {
	sequence := packLinkSequence(header.Sequence)
	if header.CLCDelta {
		sequence |= linkCLCDeltaBit
	}
	ack := packLinkSequence(header.Ack) |
		linkCompressionMarker

	var encoded [LinkHeaderSize]byte
	binary.LittleEndian.PutUint16(encoded[:linkAckOffset], sequence)
	binary.LittleEndian.PutUint16(encoded[linkAckOffset:], ack)
	return encoded
}

func packLinkSequence(sequence uint32) uint16 {
	return uint16(sequence&linkSequenceMask) |
		uint16(sequence>>16)&linkWireReliableBit
}

func IsCompressedLinkPacket(packet []byte) bool {
	return len(packet) >= LinkHeaderSize &&
		binary.LittleEndian.Uint32(packet[:LinkHeaderSize]) != protocol.QWConnectionlessSequence &&
		binary.LittleEndian.Uint16(packet[linkAckOffset:])&linkCompressionMarker != 0
}

type LinkSequences struct {
	sequence    uint32
	ack         uint32
	initialized bool
}

func (s *LinkSequences) Observe(sequence, ack uint32) {
	s.sequence = sequence
	s.ack = ack
	s.initialized = true
}

func (s *LinkSequences) DecodeHeader(packet []byte) (LinkHeader, []byte, error) {
	if !IsCompressedLinkPacket(packet) {
		return LinkHeader{}, nil, ErrNotCompressed
	}

	sequenceWire := binary.LittleEndian.Uint16(packet[:linkAckOffset])
	ackWire := binary.LittleEndian.Uint16(packet[linkAckOffset:])
	header := LinkHeader{
		Sequence: expandLinkSequence(s.sequence, sequenceWire, s.initialized),
		Ack:      expandLinkSequence(s.ack, ackWire, s.initialized),
		CLCDelta: sequenceWire&linkCLCDeltaBit != 0,
	}
	s.Observe(header.Sequence, header.Ack)
	return header, packet[LinkHeaderSize:], nil
}

func expandLinkSequence(previous uint32, wire uint16, initialized bool) uint32 {
	reliable := uint32(wire&linkWireReliableBit) << 16
	low := uint32(wire) & linkSequenceMask
	if !initialized {
		return reliable | low
	}

	previous &= ^protocol.QWSequenceReliableBit
	candidate := previous&^linkSequenceMask | low
	if candidate+linkWrapDistance < previous {
		candidate = (candidate + linkSequenceMask + 1) & protocol.QWSequenceMask
	}
	return reliable | candidate
}
