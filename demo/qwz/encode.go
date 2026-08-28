package qwz

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/osm/quake/common/context"
	"github.com/osm/quake/demo/qwd"
	"github.com/osm/quake/demo/qwz/democmd"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangeenc"
	"github.com/osm/quake/qizmo/standard"
	"github.com/osm/quake/qizmo/state"
)

func Encode(qwdData []byte, frequencies *freq.Tables) ([]byte, error) {
	if frequencies == nil {
		return nil, fmt.Errorf("nil frequency tables")
	}

	packet := state.NewPacketWithAssets(0, assets.Embedded())
	e := &encoder{
		rangeEncoder:    rangeenc.New(),
		frequencies:     frequencies,
		commandState:    democmd.NewState(),
		packet:          packet,
		tracker:         standard.NewTracker(packet),
		messageEncoder:  qizmo.NewQWDEncoder(frequencies, packet),
		packetContext:   context.New(),
		qwdData:         qwdData,
		previousCommand: exitCommand,
	}
	if err := e.encode(); err != nil {
		return nil, err
	}
	return e.rangeEncoder.Finish(), nil
}

type encoder struct {
	rangeEncoder   *rangeenc.Encoder
	frequencies    *freq.Tables
	commandState   *democmd.State
	packet         *state.Packet
	tracker        *standard.Tracker
	messageEncoder *qizmo.Encoder
	packetContext  *context.Context

	qwdData     []byte
	offset      int
	recordIndex int

	timestampMillis  uint32
	timestampDelta   byte
	timestampStarted bool
	previousCommand  byte
	sequence         uint32
	previousAckHigh  bool
	previousSeqHigh  bool
	afterDemoSet     bool
	disconnected     bool
}

func (e *encoder) encode() error {
	for e.offset < len(e.qwdData) {
		recordOffset := e.offset
		header, err := e.take(qwd.RecordHeaderSize)
		if err != nil {
			return fmt.Errorf("record %d header at offset %d: %w", e.recordIndex, recordOffset, err)
		}

		timestampBits := binary.LittleEndian.Uint32(header[:qwd.TimestampSize])
		timestamp, ok := timestampMilliseconds(timestampBits)
		if !ok {
			return fmt.Errorf(
				"record %d timestamp %v is not QWZ-representable",
				e.recordIndex,
				math.Float32frombits(timestampBits),
			)
		}
		if err := e.encodeTimestamp(timestamp); err != nil {
			return fmt.Errorf("record %d timestamp: %w", e.recordIndex, err)
		}

		command := header[qwd.TimestampSize]
		if command > protocol.DemoSet {
			return fmt.Errorf("record %d unknown QWD command %d", e.recordIndex, command)
		}
		if err := e.rangeEncoder.EncodeSymbol(demoCommandCumulative, uint32(command)); err != nil {
			return fmt.Errorf("record %d encode outer command %d: %w", e.recordIndex, command, err)
		}

		switch command {
		case protocol.DemoCmd:
			if err := e.encodeDemoCmd(); err != nil {
				return fmt.Errorf("record %d demo_cmd: %w", e.recordIndex, err)
			}
		case protocol.DemoRead:
			if err := e.encodeDemoRead(); err != nil {
				return fmt.Errorf("record %d demo_read: %w", e.recordIndex, err)
			}
		case protocol.DemoSet:
			if err := e.encodeDemoSet(); err != nil {
				return fmt.Errorf("record %d demo_set: %w", e.recordIndex, err)
			}
		}

		e.previousCommand = command
		e.recordIndex++
	}

	// A DEMO_CMD always consumes one following time symbol. At end of input
	// Qizmo emits symbol zero before the EXIT record; the resulting timestamp
	// is not observable because EXIT has no QWD record.
	if e.previousCommand == protocol.DemoCmd {
		if err := e.rangeEncoder.EncodeSymbol(e.frequencies.CumulativeRow(freq.DemoTime), 0); err != nil {
			return fmt.Errorf("encode final demo_cmd time: %w", err)
		}
	}
	if err := e.rangeEncoder.EncodeSymbol(demoCommandCumulative, exitCommand); err != nil {
		return fmt.Errorf("encode exit: %w", err)
	}
	return nil
}

func (e *encoder) encodeTimestamp(timestamp uint32) error {
	if e.previousCommand != protocol.DemoCmd {
		return nil
	}

	if !e.timestampStarted {
		e.timestampMillis = timestamp
		e.timestampStarted = true
	}

	advance := timestamp - e.timestampMillis
	if advance > math.MaxUint8 {
		advance = math.MaxUint8
	}

	nextTimestampDelta := byte(advance)
	symbol := nextTimestampDelta - e.timestampDelta
	if err := e.rangeEncoder.EncodeSymbol(e.frequencies.CumulativeRow(freq.DemoTime), uint32(symbol)); err != nil {
		return err
	}
	e.timestampDelta = nextTimestampDelta
	e.timestampMillis = timestamp
	return nil
}

func (e *encoder) encodeDemoCmd() error {
	payload, err := e.take(democmd.PayloadSize)
	if err != nil {
		return err
	}
	var command [democmd.PayloadSize]byte
	copy(command[:], payload)
	if err := democmd.Encode(e.rangeEncoder, e.frequencies, e.commandState, command); err != nil {
		return err
	}
	e.packet.CommitCommandScale(e.commandState.Msec())
	return nil
}

func (e *encoder) encodeDemoRead() error {
	sizeBytes, err := e.take(qwd.ReadSizeFieldSize)
	if err != nil {
		return err
	}
	size := binary.LittleEndian.Uint32(sizeBytes)
	if size == 0 {
		return fmt.Errorf("zero-length packet terminates the Qizmo compressor")
	}
	if size > math.MaxUint16 {
		return fmt.Errorf("raw packet size %d exceeds QWZ's %d-byte limit", size, math.MaxUint16)
	}
	payload, err := e.take(int(size))
	if err != nil {
		return err
	}
	payload, disconnect := canonicalizeConnectionPacket(
		e.packetContext,
		payload,
		e.disconnected,
	)

	previousSeq := e.sequence
	if e.previousAckHigh && !e.previousSeqHigh {
		previousSeq |= protocol.QWSequenceReliableBit
	}
	plan, supported := e.messageEncoder.PlanPacket(payload, previousSeq)
	useCompressed := !e.afterDemoSet && !disconnect && supported
	if useCompressed {
		if err := e.encodeCompressedPacket(plan); err != nil {
			return err
		}
	} else {
		if err := e.encodeRawPacket(payload); err != nil {
			return err
		}
	}

	if len(payload) >= protocol.QWServerPacketHeaderSize {
		seq := binary.LittleEndian.Uint32(payload)
		ack := binary.LittleEndian.Uint32(payload[protocol.QWPacketAckOffset:])
		if seq != protocol.QWConnectionlessSequence {
			e.previousSeqHigh = seq&protocol.QWSequenceReliableBit != 0
			e.sequence = seq
			if useCompressed {
				e.sequence &= protocol.QWSequenceMask
			}
			e.previousAckHigh = ack&protocol.QWSequenceReliableBit != 0
		}
		if useCompressed {
			plan.Commit()
		} else {
			if err := e.tracker.Observe(payload, seq); err != nil {
				return fmt.Errorf("track raw svc seq=%d: %w", seq, err)
			}
		}
	}
	if disconnect {
		e.disconnected = true
	}
	e.afterDemoSet = false
	return nil
}

func (e *encoder) encodeCompressedPacket(plan *qizmo.PacketPlan) error {
	if err := e.rangeEncoder.EncodeSymbol(
		e.frequencies.CumulativeRow(freq.DemoMode),
		uint32(plan.Mode()),
	); err != nil {
		return err
	}
	return plan.EncodeBody(e.rangeEncoder)
}

func (e *encoder) encodeRawPacket(payload []byte) error {
	if err := e.rangeEncoder.EncodeSymbol(e.frequencies.CumulativeRow(freq.DemoMode), rawPacketMode); err != nil {
		return err
	}
	size := uint32(len(payload))
	byteCumulative := e.frequencies.CumulativeRow(freq.ByteValue)
	if err := e.rangeEncoder.EncodeSymbol(byteCumulative, uint32(byte(size))); err != nil {
		return err
	}
	if err := e.rangeEncoder.EncodeSymbol(byteCumulative, uint32(byte(size>>8))); err != nil {
		return err
	}
	for _, b := range payload {
		if err := e.rangeEncoder.EncodeSymbol(byteCumulative, uint32(b)); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) encodeDemoSet() error {
	payload, err := e.take(qwd.SetPayloadSize)
	if err != nil {
		return err
	}
	byteCumulative := e.frequencies.CumulativeRow(freq.ByteValue)
	for _, b := range payload {
		if err := e.rangeEncoder.EncodeSymbol(byteCumulative, uint32(b)); err != nil {
			return err
		}
	}
	e.packet.CommandSequence = binary.LittleEndian.Uint32(payload)
	e.packet.RebuildRemaps()
	e.afterDemoSet = true
	return nil
}

func (e *encoder) take(n int) ([]byte, error) {
	if n < 0 || n > len(e.qwdData)-e.offset {
		return nil, fmt.Errorf("truncated input: need %d bytes, have %d", n, len(e.qwdData)-e.offset)
	}
	data := e.qwdData[e.offset : e.offset+n]
	e.offset += n
	return data, nil
}

func timestampMilliseconds(bits uint32) (uint32, bool) {
	timestamp := math.Float32frombits(bits)
	if math.IsNaN(float64(timestamp)) || math.IsInf(float64(timestamp), 0) {
		return 0, false
	}

	scaled := float64(timestamp)*millisecondsPerSecond + 0.5
	if scaled < math.MinInt64 || scaled > math.MaxInt64 {
		return 0, false
	}
	return uint32(int64(scaled)), true
}
