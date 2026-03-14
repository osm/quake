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

	packet     []byte
	packetEnts map[uint16]state.EntityRecord
}

// Decode is a normal QuakeWorld svc parser. It does not use the parsed data
// directly, but parsing is required to keep the decoder state in sync.
//
// This parser is likely still incomplete. If QWZ decoding fails on a sample,
// one of the first things to check is whether this parser is handling the raw
// svc stream correctly enough to keep the shared state aligned.
func Decode(packet []byte, st *state.Packet, seq uint32) error {
	if len(packet) < 8 {
		return fmt.Errorf("raw packet too short: %d", len(packet))
	}

	if seq == 0xffffffff {
		return nil
	}

	d := &decoder{
		r:      newReader(packet[8:]),
		st:     st,
		packet: packet,
	}

	d.st.BeginPacket(seq)

	for d.r.Len() > 0 {
		posBeforeOp := int(d.r.Size()) - d.r.Len()

		svcCode, err := d.r.ReadByte()
		if err != nil {
			return err
		}

		if svcCode == 0 {
			break
		}

		switch svcCode {
		case 0x01, 0x02, 0x1b, 0x1c, 0x21, 0x22, 0x23:

		case 0x03:
			_, err = d.r.ReadN(2)
		case 0x04:
			_, err = d.r.ReadN(2)
		case 0x05, 0x07, 0x18, 0x20:
			_, err = d.r.ReadN(1)

		case 0x06:
			w, e := d.r.ReadU16()
			if e != nil {
				err = e
				break
			}
			if int16(w) < 0 {
				_, err = d.r.ReadN(1)
				if err != nil {
					break
				}
			}
			if w&0x4000 != 0 {
				_, err = d.r.ReadN(1)
				if err != nil {
					break
				}
			}
			_, err = d.r.ReadN(7)

		case 0x08:
			_, err = d.r.ReadByte()
			if err == nil {
				_, err = d.r.ReadString()
			}

		case 0x09:
			var s []byte
			s, err = d.r.ReadString()
			if err == nil {
				text := string(bytes.TrimSuffix(s, []byte{0}))
				if !d.st.TablesBuilt &&
					(strings.HasPrefix(text, "cmd prespawn") ||
						strings.HasPrefix(text, "cmd spawn")) {
					d.st.RebuildRemaps()
				}
			}
		case 0x1a, 0x1f, 0x51:
			_, err = d.r.ReadString()

		case 0x0b:
			b, e := d.r.ReadN(8)
			if e != nil {
				err = e
				break
			}
			_, err = d.r.ReadString()
			if err != nil {
				break
			}
			playerNum, e := d.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			d.st.PlayerIndex = playerNum & 0x7f
			_, err = d.r.ReadString()
			if err != nil {
				break
			}
			if binary.LittleEndian.Uint32(b[0:4]) > 0x18 {
				_, err = d.r.ReadN(40)
			}

		case 0x0c:
			_, err = d.r.ReadByte()
			if err == nil {
				_, err = d.r.ReadString()
			}
		case 0x0e, 0x24:
			_, err = d.r.ReadN(3)
		case 0x10:
			_, err = d.r.ReadN(1)
		case 0x11:
			_, err = d.r.ReadN(6)
		case 0x13:
			_, err = d.r.ReadN(8)
		case 0x0a:
			_, err = d.r.ReadN(3)
		case 0x14:
			_, err = d.r.ReadN(13)

		case 0x16:
			err = d.parseSpawnBaseline()

		case 0x17:
			t, e := d.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			switch t {
			case 0x02, 0x0c:
				_, err = d.r.ReadN(7)
			case 0x05, 0x06, 0x09:
				_, err = d.r.ReadN(14)
			default:
				_, err = d.r.ReadN(6)
			}
		case 0x1d, 0x1e:
			_, err = d.r.ReadN(9)
		case 0x25, 0x26:
			_, err = d.r.ReadN(5)

		case 0x27:
			_, err = d.r.ReadN(2)

		case 0x28:
			_, err = d.r.ReadN(5)
			if err == nil {
				_, err = d.r.ReadString()
			}
		case 0x29:
			h, e := d.r.ReadN(3)
			if e != nil {
				err = e
				break
			}
			n := binary.LittleEndian.Uint16(h[0:2])
			if n != 0xffff {
				_, err = d.r.ReadN(int(n))
			}
		case 0x3e:
			return nil

		case 0x2a:
			err = d.parsePlayerInfo(seq)
		case 0x2b:
			n, e := d.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			_, err = d.r.ReadN(int(n) * 6)
		case 0x2c:
			_, err = d.r.ReadN(1)
		case 0x2d, 0x2e:
			start, e := d.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			modelIdx := int(start)
			for {
				s, e := d.r.ReadString()
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
					d.st.ModelNames[modelIdx] = name
				} else {
					d.st.SoundNames[modelIdx] = name
				}
				if svcCode == 0x2d && string(s[:len(s)-1]) == "progs/player.mdl" {
					d.st.PlayerFreqIndex = byte(modelIdx)
				}
			}
			if err != nil {
				break
			}
			_, err = d.r.ReadByte()
			if err == nil {
				posAfter := int(d.r.Size()) - d.r.Len()
				chunk := append([]byte(nil), d.packet[8+posBeforeOp:8+posAfter]...)
				if svcCode == 0x2d {
					d.st.AddModelChunk(chunk)
				} else {
					d.st.AddSoundChunk(chunk)
				}
			}
		case 0x2f:
			err = d.parsePacketEntities(nil)

		case 0x30:
			ref, e := d.r.ReadByte()
			if e != nil {
				err = e
				break
			}
			base, _ := d.st.FindRawEntitiesByte(ref)
			err = d.parsePacketEntities(base)
		case 0x31, 0x32:
			_, err = d.r.ReadN(4)

		case 0x33:
			_, err = d.r.ReadByte()
			if err == nil {
				_, err = d.r.ReadString()
			}
			if err == nil {
				_, err = d.r.ReadString()
			}

		case 0x34:
			_, err = d.r.ReadString()
			if err == nil {
				_, err = d.r.ReadString()
			}
		case 0x35:
			_, err = d.r.ReadN(2)

		case 0x52:
			_, err = d.r.ReadN(0xa2)

		case 0x53:
			_, err = d.r.ReadN(0x22)

		default:
			if svcCode >= 0x4a {
				return nil
			}
			pos := int(d.r.Size()) - d.r.Len() - 1
			return fmt.Errorf("unsupported raw svc 0x%02x at pos %d", svcCode, pos)
		}

		if err != nil {
			return fmt.Errorf("decode raw svc 0x%02x: %w", svcCode, err)
		}
	}

	histSeq := seq
	if len(d.st.CurrentPlayers) != 0 && d.st.CmdSeqNo != 0 {
		histSeq = d.st.CmdSeqNo + 1
	}

	if d.packetEnts != nil {
		for entNum, rec := range d.packetEnts {
			d.st.EntityLast[entNum] = rec
			d.st.EntityRaw[entNum] = rec
			d.st.EntityLastRaw[entNum] = true
		}

		d.st.CommitRawEntitiesByte(byte(seq), d.packetEnts)
		d.st.CommitEntities(histSeq, d.packetEnts)
	}

	d.st.CommitPacketAs(histSeq)
	return nil
}
