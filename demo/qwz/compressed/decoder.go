package compressed

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/qwz/freq"
	"github.com/osm/quake/demo/qwz/rangedec"
	"github.com/osm/quake/demo/qwz/state"
)

type decoder struct {
	rd    *rangedec.Decoder
	ft    *freq.Tables
	state *state.Packet

	lastEntityAndX      uint32
	lastYZ              uint32
	lastPlayerIndex     byte
	lastPingPlayerIndex byte
	lastPLPlayerIndex   byte

	primaryPlayerPosXY uint32
	primaryPlayerPosZ  uint32
	basePlayers        []state.PlayerRecord
	packetScale        int
}

func Decode(
	outer *rangedec.Decoder,
	ft *freq.Tables,
	st *state.Packet,
	seq, ack uint32,
) ([]byte, error) {
	rd := *outer
	decoder := &decoder{
		rd:                  &rd,
		ft:                  ft,
		state:               st,
		lastPlayerIndex:     0xff,
		lastPingPlayerIndex: 0xff,
		lastPLPlayerIndex:   0xff,
	}

	out := make([]byte, 0, 8192)

	st.BeginPacket(seq)

	out = append(out, 0x00, 0x00, 0x00, 0x00)

	out = appendUint32LE(out, seq&0x7fffffff)
	out = appendUint32LE(out, ack)

	out = append(out, 0x2a)
	body, err := decoder.decodeSVCPlayerInfo()
	if err != nil {
		return nil, err
	}
	out = append(out, body...)
	decoder.refreshPacketContext()

	for {
		svcCode, err := decoder.rd.DecodeFreqByte(decoder.ft, freq.SVCType)
		if err != nil {
			return nil, err
		}

		if svcCode == 0 {
			break
		}

		out = append(out, svcCode)

		switch svcCode {
		case 0x01, 0x02, 0x1b, 0x1c, 0x21, 0x22, 0x23:

		case 0x03:
			out, err = decoder.appendFreqBytes(out, freq.SVCUpdateStatIndex, 1)
			if err != nil {
				return nil, err
			}
			out, err = decoder.appendFreqBytes(out, freq.SVCStatValue, 1)
			if err != nil {
				return nil, err
			}

		case 0x06:
			out, err = decoder.decodeSVCSound(out)
			if err != nil {
				return nil, err
			}

		case 0x08:
			out, err = decoder.decodeSVCPrint(out)
			if err != nil {
				return nil, err
			}

		case 0x09:
			out, err = decoder.decodeSVCStufftext(out)
			if err != nil {
				return nil, err
			}

		case 0x0e:
			out, err = decoder.appendFreqBytes(out, freq.PlayerInfoSlot1, 1)
			if err != nil {
				return nil, err
			}
			out, err = decoder.appendFreqBytes(out, freq.SVCFragsPlayerSlot, 1)
			if err != nil {
				return nil, err
			}
			out, err = decoder.appendFreqBytes(out, freq.SVCFragsValue, 1)
			if err != nil {
				return nil, err
			}

		case 0x13:
			out, err = decoder.decodeSVCDamage(out)
			if err != nil {
				return nil, err
			}

		case 0x1a:
			out, err = decoder.decodeSVCCenterPrint(out)
			if err != nil {
				return nil, err
			}

		case 0x1d, 0x1e:
			out, err = decoder.appendFreqBytes(out, freq.ByteValue, 9)
			if err != nil {
				return nil, err
			}

		case 0x17:
			out, err = decoder.decodeSVCTempEntity(out)
			if err != nil {
				return nil, err
			}

		case 0x31, 0x32:
			out, err = decoder.appendFreqBytes(out, freq.ByteValue, 4)
			if err != nil {
				return nil, err
			}

		case 0x0a:
			out, err = decoder.appendFreqBytes(out, freq.ByteValue, 3)
			if err != nil {
				return nil, err
			}

		case 0x10:
			out, err = decoder.appendFreqBytes(out, freq.ByteValue, 2)
			if err != nil {
				return nil, err
			}

		case 0x18, 0x20:
			out, err = decoder.appendFreqBytes(out, freq.ByteValue, 1)
			if err != nil {
				return nil, err
			}

		case 0x27:
			lo, err := decoder.rd.DecodeFreqByte(decoder.ft, freq.SVCTEntBeamEntityLo)
			if err != nil {
				return nil, err
			}
			hi, err := decoder.rd.DecodeFreqByte(decoder.ft, freq.SVCTEntBeamEntityHi)
			if err != nil {
				return nil, err
			}
			id := uint16(decoder.lastEntityAndX>>16) ^ uint16(lo) ^ (uint16(hi) << 8)
			decoder.lastEntityAndX =
				(decoder.lastEntityAndX & 0x0000ffff) | (uint32(id) << 16)
			out = appendUint16LE(out, id)

		case 0x28:
			out, err = decoder.appendFreqBytes(out, freq.PlayerInfoSlot1, 1)
			if err != nil {
				return nil, err
			}
			out, err = decoder.appendFreqBytes(out, freq.SVCUpdateUserInfoUserID, 4)
			if err != nil {
				return nil, err
			}
			out, err = decoder.decodeString(out, freq.SVCUpdateUserInfoString)
			if err != nil {
				return nil, err
			}

		case 0x0c:
			out, err = decoder.appendFreqBytes(out, freq.ByteValue, 1)
			if err != nil {
				return nil, err
			}
			out, err = decoder.decodeString(out, freq.ByteValue)
			if err != nil {
				return nil, err
			}

		case 0x2c:
			out, err = decoder.appendFreqBytes(out, freq.SVCChokeCount, 1)
			if err != nil {
				return nil, err
			}

		case 0x34:
			out, err = decoder.decodeString(out, freq.SVCServerInfoString)
			if err != nil {
				return nil, err
			}
			out, err = decoder.decodeString(out, freq.SVCServerInfoString)
			if err != nil {
				return nil, err
			}

		case 0x33:
			out, err = decoder.decodeSVCSetInfo(out)
			if err != nil {
				return nil, err
			}

		case 0x2b:
			out, err = decoder.decodeSVCNails(out)
			if err != nil {
				return nil, err
			}

		case 0x24:
			out, err = decoder.decodeSVCUpdatePing(out)
			if err != nil {
				return nil, err
			}

		case 0x35:
			out, err = decoder.decodeSVCUpdatePL(out)
			if err != nil {
				return nil, err
			}

		case 0x53:
			out, err = decoder.decodeSVCQizmoVoice(out)
			if err != nil {
				return nil, err
			}

		case 0x25:
			out, err = decoder.appendFreqBytes(out, freq.ByteValue, 5)
			if err != nil {
				return nil, err
			}

		case 0x26:
			for _, freqTableAddr := range []uint32{
				freq.SVCUpdateStatLongIndex,
				freq.SVCUpdateStatLongByte0,
				freq.SVCUpdateStatLongByte1,
				freq.SVCUpdateStatLongByte2,
				freq.SVCUpdateStatLongByte3,
			} {
				out, err = decoder.appendFreqBytes(out, freqTableAddr, 1)
				if err != nil {
					return nil, err
				}
			}

		case 0x2a:
			out, err = decoder.decodeSVCPlayerInfoDeltas(out)
			if err != nil {
				return nil, err
			}

		case 0x2f:
			body, err := decoder.decodeSVCPacketEntitiesFull()
			if err != nil {
				return nil, err
			}
			out = append(out, body...)

		case 0x30:
			body, err := decoder.decodeSVCPacketEntitiesFull()
			if err != nil {
				return nil, err
			}
			out[len(out)-1] = 0x2f
			out = append(out, body...)

		default:
			return nil, fmt.Errorf("unsupported svc opcode 0x%02x", svcCode)
		}
	}

	st.CommitPacket()
	*outer = rd
	binary.LittleEndian.PutUint32(out[:4], uint32(len(out)-4))

	return out, nil
}

func (d *decoder) refreshPacketContext() {
	if d.state.NumCurrentPlayers() != 0 {
		lastRec := d.state.LastCurrentPlayer()
		d.lastEntityAndX = uint32(uint16(lastRec[1] & 0xffff))
		d.lastYZ = (lastRec[1] & 0xffff0000) | uint32(uint16(lastRec[2]&0xffff))

		firstRec := d.state.FirstCurrentPlayer()
		d.primaryPlayerPosXY = firstRec[1]
		d.primaryPlayerPosZ = firstRec[2]
	}

	if d.state.PacketHasBase {
		if h, ok := d.state.PlayerHistoryEntry(d.state.PacketBaseSeq); ok {
			d.basePlayers = h.Recs
		} else {
			d.basePlayers = nil
		}
		d.packetScale =
			int(d.state.Scale(d.state.PacketBaseSeq)) *
				int(d.state.SeqNo()-d.state.PacketBaseSeq)
	} else {
		d.basePlayers = nil
		d.packetScale = 0
	}
}

func (d *decoder) appendFreqBytes(
	out []byte,
	freqTableAddr uint32,
	count int,
) ([]byte, error) {
	for i := 0; i < count; i++ {
		b, err := d.rd.DecodeFreqByte(d.ft, freqTableAddr)
		if err != nil {
			return nil, err
		}

		out = append(out, b)
	}

	return out, nil
}
