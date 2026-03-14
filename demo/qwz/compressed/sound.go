package compressed

import (
	"encoding/binary"
	"github.com/osm/quake/demo/qwz/freq"
	"github.com/osm/quake/demo/qwz/state"
)

func (d *decoder) decodeSVCSound(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	lastEntityAndX := &d.lastEntityAndX
	lastYZ := &d.lastYZ
	basePlayers := d.basePlayers

	soundPacket := make([]byte, 0, 9)
	b, err := rd.DecodeFreqByte(ft, freq.SVCSoundChannelLo)
	if err != nil {
		return nil, err
	}
	soundPacket = append(soundPacket, byte((byte(*lastEntityAndX>>16)<<3)^b))
	b, err = rd.DecodeFreqByte(ft, freq.SVCSoundChannelHi)
	if err != nil {
		return nil, err
	}
	soundPacket = append(soundPacket, byte((byte(uint16(*lastEntityAndX>>16)>>5))^b))
	channelEnt := binary.LittleEndian.Uint16(soundPacket[:2])
	ent := uint16((channelEnt >> 3) & 0x01ff)
	*lastEntityAndX = (*lastEntityAndX & 0x0000ffff) | (uint32(ent) << 16)
	if int16(channelEnt) < 0 {
		b, err = rd.DecodeFreqByte(ft, freq.SVCSoundVolume)
		if err != nil {
			return nil, err
		}
		soundPacket = append(soundPacket, b)
	}
	if channelEnt&0x4000 != 0 {
		b, err = rd.DecodeFreqByte(ft, freq.SVCSoundAttenuation)
		if err != nil {
			return nil, err
		}
		soundPacket = append(soundPacket, b)
	}
	soundNum, err := rd.DecodeFreqByte(ft, freq.SVCSoundIndex)
	if err != nil {
		return nil, err
	}
	soundPrecacheIdx := st.SoundRemapByte(soundNum)
	soundPacket = append(soundPacket, soundPrecacheIdx)
	deltaFlags, err := rd.DecodeFreqByte(ft, freq.SVCSoundFlags)
	if err != nil {
		return nil, err
	}

	coordX := uint16(*lastEntityAndX & 0xffff)
	coordY := uint16((*lastYZ >> 16) & 0xffff)
	coordZ := uint16(*lastYZ & 0xffff)

	if int8(deltaFlags) >= 0 && ent != 0 {
		if ent < 0x21 {
			if len(basePlayers) != 0 {
				list := st.CurrentPlayers
				if deltaFlags&0x40 != 0 {
					list = basePlayers
				}
				if x, y, z, ok := resolvePlayerCoords(list, ent); ok {
					coordX, coordY, coordZ = x, y, z
				}
			}
		} else {
			found := false
			stoppedBefore := false
			if ents, ok := st.FindEntHistoryBySeq(st.SeqNo()); ok {
				for _, rec := range ents {
					e := state.EntityNumber(rec)
					if e == ent {
						coordX = state.EntityRecordU16(rec, 12)
						coordY = state.EntityRecordU16(rec, 14)
						coordZ = state.EntityRecordU16(rec, 16)
						found = true
						break
					}
					if ent < e {
						stoppedBefore = true
						break
					}
				}
			}
			if !found && stoppedBefore {
				if rec, ok := st.Baselines[ent]; ok {
					if state.EntityRecordU16(rec, 12) != 0 ||
						state.EntityRecordU16(rec, 14) != 0 ||
						state.EntityRecordU16(rec, 16) != 0 {
						coordX = state.EntityRecordU16(rec, 12)
						coordY = state.EntityRecordU16(rec, 14)
						coordZ = state.EntityRecordU16(rec, 16)
					}
				}
			}
		}
	}

	if deltaFlags&0x01 != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaXLo)
		if err != nil {
			return nil, err
		}
		coordX = uint16(int16(coordX) + int16(int8(v)))
	}
	if deltaFlags&0x02 != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaXHi)
		if err != nil {
			return nil, err
		}
		coordX = uint16(int16(coordX) + int16(uint16(v)<<8))
	}
	soundPacket = appendUint16LE(soundPacket, coordX)

	if deltaFlags&0x04 != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaYLo)
		if err != nil {
			return nil, err
		}
		coordY = uint16(int16(coordY) + int16(int8(v)))
	}
	if deltaFlags&0x08 != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaYHi)
		if err != nil {
			return nil, err
		}
		coordY = uint16(int16(coordY) + int16(uint16(v)<<8))
	}
	soundPacket = appendUint16LE(soundPacket, coordY)

	if deltaFlags&0x10 != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaZLo)
		if err != nil {
			return nil, err
		}
		coordZ = uint16(int16(coordZ) + int16(int8(v)))
	}
	if deltaFlags&0x20 != 0 {
		v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaZHi)
		if err != nil {
			return nil, err
		}
		coordZ = uint16(int16(coordZ) + int16(uint16(v)<<8))
	}
	soundPacket = appendUint16LE(soundPacket, coordZ)
	*lastEntityAndX = (uint32(ent) << 16) | uint32(coordX)
	*lastYZ = (uint32(coordY) << 16) | uint32(coordZ)
	return append(out, soundPacket...), nil
}
