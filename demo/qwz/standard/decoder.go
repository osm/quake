package standard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/osm/quake/demo/qwz/state"
)

type decoder struct {
	r  *reader
	st *state.Packet

	handlers map[Event][]HandlerFunc

	packet     []byte
	packetEnts map[uint16]state.EntityRecord
}

type Event int

const (
	QizmoVoice Event = iota
)

type HandlerFunc func(payload []byte)

type Decoder struct {
	state    *state.Packet
	handlers map[Event][]HandlerFunc
}

// New returns a normal QuakeWorld svc parser. It does not use the parsed data
// directly, but parsing is required to keep the decoder state in sync.
//
// This parser is likely still incomplete. If QWZ decoding fails on a sample,
// one of the first things to check is whether this parser is handling the raw
// svc stream correctly enough to keep the shared state aligned.
func New(st *state.Packet) *Decoder {
	return &Decoder{
		state:    st,
		handlers: make(map[Event][]HandlerFunc),
	}
}

func (d *Decoder) HandleFunc(event Event, fn HandlerFunc) {
	d.handlers[event] = append(d.handlers[event], fn)
}

func (d *Decoder) Decode(packet []byte, seq uint32) error {
	if len(packet) < 8 {
		return fmt.Errorf("raw packet too short: %d", len(packet))
	}

	if seq == 0xffffffff {
		return nil
	}

	parser := &decoder{
		r:        newReader(packet[8:]),
		st:       d.state,
		handlers: d.handlers,
		packet:   packet,
	}

	parser.st.BeginPacket(seq)

	for parser.r.Len() > 0 {
		posBeforeOp := int(parser.r.Size()) - parser.r.Len()

		svcCode, err := parser.r.ReadByte()
		if err != nil {
			return err
		}

		if svcCode == 0 {
			break
		}

		switch svcCode {
		case 0x01, 0x02, 0x1b, 0x1c, 0x21, 0x22, 0x23:

		case 0x03:
			_, err = parser.r.ReadN(2)
		case 0x04:
			_, err = parser.r.ReadN(2)
		case 0x05, 0x07, 0x18, 0x20:
			_, err = parser.r.ReadN(1)

		case 0x06:
			w, e := parser.r.ReadU16()
			if e != nil {
				err = e
				break
			}
			if int16(w) < 0 {
				_, err = parser.r.ReadN(1)
				if err != nil {
					break
				}
			}
			if w&0x4000 != 0 {
				_, err = parser.r.ReadN(1)
				if err != nil {
					break
				}
			}
			_, err = parser.r.ReadN(7)

		case 0x08:
			_, err = parser.r.ReadByte()
			if err == nil {
				_, err = parser.r.ReadString()
			}

		case 0x09:
			var s []byte
			s, err = parser.r.ReadString()
			if err == nil {
				text := string(bytes.TrimSuffix(s, []byte{0}))
				if !parser.st.TablesBuilt &&
					(strings.HasPrefix(text, "cmd prespawn") ||
						strings.HasPrefix(text, "cmd spawn")) {
					parser.st.RebuildRemaps()
				}
			}
		case 0x1a, 0x1f, 0x51:
			_, err = parser.r.ReadString()

		case 0x0b:
			b, e := parser.r.ReadN(8)
			if e != nil {
				err = e
				break
			}
			_, err = parser.r.ReadString()
			if err != nil {
				break
			}
			playerNum, e := parser.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			parser.st.PlayerIndex = playerNum & 0x7f
			_, err = parser.r.ReadString()
			if err != nil {
				break
			}
			if binary.LittleEndian.Uint32(b[0:4]) > 0x18 {
				_, err = parser.r.ReadN(40)
			}

		case 0x0c:
			_, err = parser.r.ReadByte()
			if err == nil {
				_, err = parser.r.ReadString()
			}
		case 0x0e, 0x24:
			_, err = parser.r.ReadN(3)
		case 0x10:
			_, err = parser.r.ReadN(1)
		case 0x11:
			_, err = parser.r.ReadN(6)
		case 0x13:
			_, err = parser.r.ReadN(8)
		case 0x0a:
			_, err = parser.r.ReadN(3)
		case 0x14:
			_, err = parser.r.ReadN(13)

		case 0x16:
			err = parser.parseSpawnBaseline()

		case 0x17:
			t, e := parser.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			switch t {
			case 0x02, 0x0c:
				_, err = parser.r.ReadN(7)
			case 0x05, 0x06, 0x09:
				_, err = parser.r.ReadN(14)
			default:
				_, err = parser.r.ReadN(6)
			}
		case 0x1d, 0x1e:
			_, err = parser.r.ReadN(9)
		case 0x25, 0x26:
			_, err = parser.r.ReadN(5)

		case 0x27:
			_, err = parser.r.ReadN(2)

		case 0x28:
			_, err = parser.r.ReadN(5)
			if err == nil {
				_, err = parser.r.ReadString()
			}
		case 0x29:
			h, e := parser.r.ReadN(3)
			if e != nil {
				err = e
				break
			}
			n := binary.LittleEndian.Uint16(h[0:2])
			if n != 0xffff {
				_, err = parser.r.ReadN(int(n))
			}
		case 0x3e:
			return nil

		case 0x2a:
			err = parser.parsePlayerInfo(seq)
		case 0x2b:
			n, e := parser.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			_, err = parser.r.ReadN(int(n) * 6)
		case 0x2c:
			_, err = parser.r.ReadN(1)
		case 0x2d, 0x2e:
			start, e := parser.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			if svcCode == 0x2d && start == 0 {
				parser.st.ResetEntityTracking()
			}
			modelIdx := int(start)
			for {
				s, e := parser.r.ReadString()
				if e != nil {
					err = e
					break
				}
				if len(s) == 1 {
					break
				}
				modelIdx++
				name := string(s[:len(s)-1])
				if svcCode == 0x2d {
					parser.st.ModelNames[modelIdx] = name
				} else {
					parser.st.SoundNames[modelIdx] = name
				}
				if svcCode == 0x2d && string(s[:len(s)-1]) == "progs/player.mdl" {
					parser.st.PlayerFreqIndex = byte(modelIdx)
				}
			}
			if err != nil {
				break
			}
			_, err = parser.r.ReadByte()
			if err == nil {
				posAfter := int(parser.r.Size()) - parser.r.Len()
				chunk := append([]byte(nil), parser.packet[8+posBeforeOp:8+posAfter]...)
				if svcCode == 0x2d {
					parser.st.AddModelChunk(chunk)
				} else {
					parser.st.AddSoundChunk(chunk)
				}
			}
		case 0x2f:
			err = parser.parsePacketEntities(nil)

		case 0x30:
			ref, e := parser.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			base, _ := parser.st.FindRawEntitiesByte(ref)
			err = parser.parsePacketEntities(base)
		case 0x31, 0x32:
			_, err = parser.r.ReadN(4)

		case 0x33:
			_, err = parser.r.ReadByte()
			if err == nil {
				_, err = parser.r.ReadString()
			}
			if err == nil {
				_, err = parser.r.ReadString()
			}

		case 0x34:
			_, err = parser.r.ReadString()
			if err == nil {
				_, err = parser.r.ReadString()
			}
		case 0x35:
			_, err = parser.r.ReadN(2)

		case 0x52:
			_, err = parser.r.ReadN(0xa2)

		case 0x53:
			payload, e := parser.r.ReadN(0x22)
			if e != nil {
				err = e
				break
			}

			for _, handler := range parser.handlers[QizmoVoice] {
				handler(append([]byte(nil), payload...))
			}

		default:
			if svcCode >= 0x4a {
				parser.r = newReader(nil)
				break
			}
			pos := int(parser.r.Size()) - parser.r.Len() - 1
			return fmt.Errorf("unsupported raw svc 0x%02x at pos %d", svcCode, pos)
		}

		if err != nil {
			return fmt.Errorf("decode raw svc 0x%02x: %w", svcCode, err)
		}
	}

	histSeq := seq
	if len(parser.st.CurrentPlayers) != 0 && parser.st.CmdSeqNo != 0 {
		histSeq = parser.st.CmdSeqNo + 1
	}

	if parser.packetEnts != nil {
		for entNum, rec := range parser.packetEnts {
			parser.st.EntityLast[entNum] = rec
			parser.st.EntityRaw[entNum] = rec
			parser.st.EntityLastRaw[entNum] = true
		}

		parser.st.CommitRawEntitiesByte(byte(seq), parser.packetEnts)
		parser.st.CommitEntities(histSeq, parser.packetEnts)
	}

	parser.st.CommitPacketAs(histSeq)
	return nil
}
