package qwz

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/osm/quake/demo/qwd"
	"github.com/osm/quake/demo/qwz/democmd"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo"
	"github.com/osm/quake/qizmo/assets"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangedec"
	"github.com/osm/quake/qizmo/standard"
	"github.com/osm/quake/qizmo/state"
)

var demoCommandCumulative = []uint32{
	0x80000000,
	0xfffffc00,
	0xfffffe00,
	0xffffffff,
}

const (
	exitCommand           = 3
	rawPacketMode         = 0
	millisecondsPerSecond = 1000
)

type decoder struct {
	rangeDecoder   *rangedec.Decoder
	frequencies    *freq.Tables
	packet         *state.Packet
	commandState   *democmd.State
	tracker        *standard.Tracker
	messageDecoder *qizmo.Decoder

	timestampMillis uint32
	timestampDelta  byte
	recordIndex     int

	sequence uint32

	out []byte
}

func Decode(qwzData []byte, frequencies *freq.Tables, packetAssets assets.Assets) ([]byte, error) {
	if frequencies == nil {
		return nil, fmt.Errorf("nil frequency tables")
	}

	rangeDecoder := rangedec.NewPadded(qwzData)

	packet := state.NewPacketWithAssets(0, packetAssets)

	d := &decoder{
		rangeDecoder: rangeDecoder,
		frequencies:  frequencies,
		packet:       packet,
		commandState: democmd.NewState(),
	}
	d.tracker = standard.NewTracker(packet)
	d.messageDecoder = qizmo.NewDecoder(rangeDecoder, frequencies, packet)

	for {
		commandSymbol, err := d.rangeDecoder.DecodeSymbol(demoCommandCumulative, uint32(len(demoCommandCumulative)))
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode demo command: %w", err)
		}

		command := byte(commandSymbol)
		if command == exitCommand {
			break
		}

		timestamp := float32(float64(d.timestampMillis) / millisecondsPerSecond)
		d.out = binary.LittleEndian.AppendUint32(d.out, math.Float32bits(timestamp))
		d.out = append(d.out, command)

		switch command {
		case protocol.DemoCmd:
			if err := d.decodeDemoCmd(); err != nil {
				return nil, fmt.Errorf("record %d demo_cmd: %w", d.recordIndex, err)
			}
		case protocol.DemoRead:
			if err := d.decodeDemoRead(); err != nil {
				if errors.Is(err, io.EOF) {
					return d.out, nil
				}
				return nil, fmt.Errorf("record %d demo_read: %w", d.recordIndex, err)
			}
		case protocol.DemoSet:
			if err := d.decodeDemoSet(); err != nil {
				return nil, fmt.Errorf("record %d demo_set: %w", d.recordIndex, err)
			}
		default:
			return nil, fmt.Errorf("unknown demo command %d at record %d", command, d.recordIndex)
		}

		d.recordIndex++
	}

	return d.out, nil
}

func (d *decoder) decodeDemoCmd() error {
	payload, err := democmd.Decode(d.rangeDecoder, d.frequencies, d.commandState)
	if err != nil {
		return err
	}
	d.out = append(d.out, payload[:]...)
	d.packet.CommitCommandScale(d.commandState.Msec())

	symbol, err := d.rangeDecoder.DecodeSymbol(
		d.frequencies.CumulativeRow(freq.DemoTime),
		freq.Symbols,
	)
	if err != nil {
		return err
	}
	d.timestampDelta += byte(symbol)
	d.timestampMillis += uint32(d.timestampDelta)
	return nil
}

func (d *decoder) decodeDemoRead() error {
	mode, err := d.rangeDecoder.DecodeSymbol(
		d.frequencies.CumulativeRow(freq.DemoMode),
		freq.Symbols,
	)
	if err != nil {
		return err
	}

	if mode == rawPacketMode {
		return d.decodeRawPacket()
	}
	return d.decodeCompressedPacket(mode)
}

func (d *decoder) decodeRawPacket() error {
	byteCumulative := d.frequencies.CumulativeRow(freq.ByteValue)

	low, err := d.rangeDecoder.DecodeSymbol(byteCumulative, freq.Symbols)
	if err != nil {
		return err
	}

	high, err := d.rangeDecoder.DecodeSymbol(byteCumulative, freq.Symbols)
	if err != nil {
		return err
	}

	size := int(low | high<<8)
	d.out = binary.LittleEndian.AppendUint32(d.out, uint32(size))

	payload := make([]byte, size)
	for i := range payload {
		symbol, err := d.rangeDecoder.DecodeSymbol(byteCumulative, freq.Symbols)
		if err != nil {
			return err
		}
		payload[i] = byte(symbol)
	}
	d.out = append(d.out, payload...)

	if len(payload) >= protocol.QWServerPacketHeaderSize {
		seq := binary.LittleEndian.Uint32(payload)
		if seq != protocol.QWConnectionlessSequence {
			d.sequence = seq
		}
		if err := d.tracker.Observe(payload, seq); err != nil {
			return fmt.Errorf("decode raw svc seq=%d: %w", seq, err)
		}
	}
	return nil
}

func (d *decoder) decodeCompressedPacket(mode uint32) error {
	d.sequence = (d.sequence + mode) & protocol.QWSequenceMask

	payload, err := d.messageDecoder.DecodePacketWithOptions(d.sequence, d.sequence, qizmo.DecodingOptions{
		TreatPrimaryOnlyPacketAsDropped: mode > 1,
	})
	if err != nil {
		return err
	}

	d.out = binary.LittleEndian.AppendUint32(d.out, uint32(len(payload)))
	d.out = append(d.out, payload...)

	if d.messageDecoder.EndedOnDroppedPacket() {
		return io.EOF
	}

	return nil
}

func (d *decoder) decodeDemoSet() error {
	byteCumulative := d.frequencies.CumulativeRow(freq.ByteValue)
	var payload [qwd.SetPayloadSize]byte

	for i := range payload {
		symbol, err := d.rangeDecoder.DecodeSymbol(byteCumulative, freq.Symbols)
		if err != nil {
			return err
		}

		payload[i] = byte(symbol)
	}

	d.out = append(d.out, payload[:]...)

	d.packet.CommandSequence = binary.LittleEndian.Uint32(payload[:])
	d.packet.RebuildRemaps()

	return nil
}
