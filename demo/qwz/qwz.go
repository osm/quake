package qwz

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/osm/quake/demo/qwz/assets"
	"github.com/osm/quake/demo/qwz/compressed"
	"github.com/osm/quake/demo/qwz/democmd"
	"github.com/osm/quake/demo/qwz/freq"
	"github.com/osm/quake/demo/qwz/rangedec"
	"github.com/osm/quake/demo/qwz/standard"
	"github.com/osm/quake/demo/qwz/state"
)

var demoCommandCumulative = []uint32{
	0x80000000,
	0xfffffc00,
	0xfffffe00,
	0xffffffff,
}

const (
	qwzDemoCmd  = 0
	qwzDemoRead = 1
	qwzDemoSet  = 2
	qwzDemoExit = 3
)

type decoder struct {
	rd      *rangedec.Decoder
	ft      *freq.Tables
	packet  *state.Packet
	demoCmd *democmd.DemoCmd
	std     *standard.Decoder
	comp    *compressed.Decoder

	qwdTimestamp     uint32
	currentTimestamp byte
	record           int

	seq uint32
	ack uint32

	out *bytes.Buffer
}

func Decode(qwzData []byte, ft *freq.Tables, assets assets.Assets) ([]byte, error) {
	paddedData := make([]byte, len(qwzData)+4)
	copy(paddedData, qwzData)

	rd, err := rangedec.New(paddedData)
	if err != nil {
		return nil, fmt.Errorf("create range decoder: %w", err)
	}

	packet := state.NewPacket(0)
	packet.CenterPrintStrings = assets.CenterPrintStrings
	packet.PrintStrings = assets.PrintStrings
	packet.PrintMode3Strings = assets.PrintMode3Strings
	packet.StuffTextStrings = assets.StuffTextStrings
	packet.SetInfoStrings = assets.SetInfoStrings
	packet.PrecacheModels = assets.PrecacheModels
	packet.PrecacheSounds = assets.PrecacheSounds

	d := &decoder{
		rd:      rd,
		ft:      ft,
		packet:  packet,
		demoCmd: &democmd.DemoCmd{Impulse: 8},
		out:     &bytes.Buffer{},
	}
	d.std = standard.New(packet)
	d.comp = compressed.New(rd, ft, packet)

	for {
		cmdSym, err := d.rd.DecodeSymbol(demoCommandCumulative, 4)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode demo command: %w", err)
		}

		qizmoCmd := byte(cmdSym)
		if qizmoCmd == qwzDemoExit {
			break
		}

		timestamp := float32(0.001 * float64(d.qwdTimestamp))
		if err := binary.Write(d.out, binary.LittleEndian, timestamp); err != nil {
			return nil, err
		}
		if _, err := d.out.Write([]byte{qizmoCmd}); err != nil {
			return nil, err
		}

		switch qizmoCmd {
		case qwzDemoCmd:
			if err := d.decodeDemoCmd(); err != nil {
				return nil, fmt.Errorf(
					"record %d demo_cmd: %w",
					d.record,
					err,
				)
			}
		case qwzDemoRead:
			if err := d.decodeDemoRead(); err != nil {
				if err == io.EOF {
					return d.out.Bytes(), nil
				}
				return nil, fmt.Errorf(
					"record %d demo_read: %w",
					d.record,
					err,
				)
			}
		case qwzDemoSet:
			if err := d.decodeDemoSet(); err != nil {
				return nil, fmt.Errorf(
					"record %d demo_set: %w",
					d.record,
					err,
				)
			}
		default:
			return nil, fmt.Errorf("unknown demo command %d at record %d", qizmoCmd, d.record)
		}

		d.record++

	}

	return d.out.Bytes(), nil
}

func (d *decoder) decodeDemoCmd() error {
	payload, err := democmd.Decode(d.rd, d.ft, d.demoCmd)
	if err != nil {
		return fmt.Errorf("decode demo command at record %d: %w", d.record, err)
	}
	if _, err := d.out.Write(payload[:]); err != nil {
		return err
	}
	d.packet.CommitCmdScale(d.demoCmd.Msec)

	s, err := d.rd.DecodeSymbol(d.ft.CumulativeRow(freq.DemoTime), 0x100)
	if err != nil {
		return fmt.Errorf("decode time delta: %w", err)
	}
	d.currentTimestamp += byte(s)
	d.qwdTimestamp += uint32(d.currentTimestamp)
	return nil
}

func (d *decoder) decodeDemoRead() error {
	mode, err := d.rd.DecodeSymbol(d.ft.CumulativeRow(freq.DemoMode), 0x100)
	if err != nil {
		return fmt.Errorf("decode mode: %w", err)
	}

	if mode == 0 {
		return d.decodeStandardSVC()
	}
	return d.decodeCompressedSVC(mode)
}

func (d *decoder) decodeStandardSVC() error {
	byteCum := d.ft.CumulativeRow(freq.ByteValue)

	lo, err := d.rd.DecodeSymbol(byteCum, 0x100)
	if err != nil {
		return fmt.Errorf("decode packet size low byte: %w", err)
	}

	hi, err := d.rd.DecodeSymbol(byteCum, 0x100)
	if err != nil {
		return fmt.Errorf("decode packet size high byte: %w", err)
	}

	size := uint32(lo) | (uint32(hi) << 8)
	if err := binary.Write(d.out, binary.LittleEndian, size); err != nil {
		return err
	}

	payload := make([]byte, size)
	for i := uint32(0); i < size; i++ {
		b, err := d.rd.DecodeSymbol(byteCum, 0x100)
		if err != nil {
			return fmt.Errorf("decode packet byte %d/%d: %w", i, size, err)
		}
		payload[i] = byte(b)
	}
	if _, err := d.out.Write(payload); err != nil {
		return err
	}

	if len(payload) >= 8 {
		seq := uint32(binary.LittleEndian.Uint32(payload[0:4]))
		ack := uint32(binary.LittleEndian.Uint32(payload[4:8]))
		if seq != 0xffffffff {
			d.seq = seq
			d.ack = ack
		}
		if err := d.std.Decode(payload, seq); err != nil {
			return fmt.Errorf("decode raw svc seq=%d: %w", seq, err)
		}
	}
	return nil
}

func (d *decoder) decodeCompressedSVC(mode uint32) error {
	d.seq = (d.seq + uint32(mode)) & 0x7fffffff
	d.ack = d.seq

	payload, err := d.comp.Decode(d.seq, d.ack, mode > 1)
	if err != nil {
		return err
	}

	if _, err := d.out.Write(payload); err != nil {
		return err
	}

	if d.comp.EndOfStreamDroppedPacket() {
		return io.EOF
	}

	return nil
}

func (d *decoder) decodeDemoSet() error {
	byteCum := d.ft.CumulativeRow(freq.ByteValue)
	var b [8]byte

	for i := 0; i < 8; i++ {
		v, err := d.rd.DecodeSymbol(byteCum, 0x100)
		if err != nil {
			return fmt.Errorf("decode demo set byte %d: %w", i, err)
		}

		b[i] = byte(v)
	}

	if _, err := d.out.Write(b[:]); err != nil {
		return err
	}

	d.packet.CmdSeqNo = binary.LittleEndian.Uint32(b[0:4])
	d.packet.RebuildRemaps()

	return nil
}
