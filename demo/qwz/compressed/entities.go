package compressed

import (
	"github.com/osm/quake/demo/qwz/freq"
	"github.com/osm/quake/demo/qwz/state"
)

func (d *decoder) decodeSVCTempEntity(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	lastEntityAndX := &d.lastEntityAndX
	lastYZ := &d.lastYZ
	basePlayers := d.basePlayers

	tempEntityType, err := rd.DecodeFreqByte(ft, freq.SVCTEntType)
	if err != nil {
		return nil, err
	}
	out = append(out, tempEntityType)

	impactX := uint16(*lastEntityAndX & 0xffff)
	impactY := uint16((*lastYZ >> 16) & 0xffff)
	impactZ := uint16(*lastYZ & 0xffff)

	switch tempEntityType {
	case 0x02, 0x0c:
		out, err = d.appendFreqBytes(out, freq.SVCTEntCount, 1)
		if err != nil {
			return nil, err
		}
	case 0x05, 0x06, 0x09:
		flags, err := rd.DecodeFreqByte(ft, freq.SVCTEntBeamFlags)
		if err != nil {
			return nil, err
		}
		if flags&0x40 != 0 {
			lo, err := rd.DecodeFreqByte(ft, freq.SVCTEntBeamEntityLo)
			if err != nil {
				return nil, err
			}
			hi, err := rd.DecodeFreqByte(ft, freq.SVCTEntBeamEntityHi)
			if err != nil {
				return nil, err
			}
			ent := uint16(*lastEntityAndX>>16) ^ uint16(lo) ^ (uint16(hi) << 8)
			*lastEntityAndX = (*lastEntityAndX & 0x0000ffff) | (uint32(ent) << 16)
		}
		ent := uint16(*lastEntityAndX >> 16)
		out = appendUint16LE(out, ent)

		if int8(flags) >= 0 && ent != 0 {
			if ent < 0x21 {
				if len(basePlayers) != 0 {
					if x, y, z, ok := resolvePlayerCoords(st.CurrentPlayers, ent); ok {
						impactX, impactY, impactZ = x, y, z
					}
				}
			} else {
				x, y, z, found, stoppedBefore := resolveEntityCoords(st, ent)
				if found {
					impactX, impactY, impactZ = x, y, z
				} else if stoppedBefore {
					if rec, ok := st.Baselines[ent]; ok {
						impactX = state.EntityRecordU16(rec, 12)
						impactY = state.EntityRecordU16(rec, 14)
						impactZ = state.EntityRecordU16(rec, 16)
					}
				}
			}
		}
		if flags&0x01 != 0 {
			v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaXLo)
			if err != nil {
				return nil, err
			}
			impactX = uint16(int16(impactX) + int16(int8(v)))
		}
		if flags&0x02 != 0 {
			v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaXHi)
			if err != nil {
				return nil, err
			}
			impactX = uint16(int16(impactX) + int16(uint16(v)<<8))
		}
		out = appendUint16LE(out, impactX)

		if flags&0x04 != 0 {
			v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaYLo)
			if err != nil {
				return nil, err
			}
			impactY = uint16(int16(impactY) + int16(int8(v)))
		}
		if flags&0x08 != 0 {
			v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaYHi)
			if err != nil {
				return nil, err
			}
			impactY = uint16(int16(impactY) + int16(uint16(v)<<8))
		}
		out = appendUint16LE(out, impactY)

		if flags&0x10 != 0 {
			v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaZLo)
			if err != nil {
				return nil, err
			}
			impactZ = uint16(int16(impactZ) + int16(int8(v)))
		}
		if flags&0x20 != 0 {
			v, err := rd.DecodeFreqByte(ft, freq.CoordDeltaZHi)
			if err != nil {
				return nil, err
			}
			impactZ = uint16(int16(impactZ) + int16(uint16(v)<<8))
		}
		out = appendUint16LE(out, impactZ)
	}

	v, err := rd.DecodeFreqByte(ft, freq.SVCTEntCoordXLoDelta)
	if err != nil {
		return nil, err
	}
	impactX = uint16(int16(impactX) + int16(int8(v)))
	v, err = rd.DecodeFreqByte(ft, freq.SVCTEntCoordXHiDelta)
	if err != nil {
		return nil, err
	}
	impactX = uint16(int16(impactX) + int16(uint16(v)<<8))
	out = appendUint16LE(out, impactX)

	v, err = rd.DecodeFreqByte(ft, freq.SVCTEntCoordYLoDelta)
	if err != nil {
		return nil, err
	}
	impactY = uint16(int16(impactY) + int16(int8(v)))
	v, err = rd.DecodeFreqByte(ft, freq.SVCTEntCoordYHiDelta)
	if err != nil {
		return nil, err
	}
	impactY = uint16(int16(impactY) + int16(uint16(v)<<8))
	out = appendUint16LE(out, impactY)

	v, err = rd.DecodeFreqByte(ft, freq.SVCTEntCoordZLoDelta)
	if err != nil {
		return nil, err
	}
	impactZ = uint16(int16(impactZ) + int16(int8(v)))
	v, err = rd.DecodeFreqByte(ft, freq.SVCTEntCoordZHiDelta)
	if err != nil {
		return nil, err
	}
	impactZ = uint16(int16(impactZ) + int16(uint16(v)<<8))
	out = appendUint16LE(out, impactZ)

	*lastEntityAndX = (*lastEntityAndX & 0xffff0000) | uint32(impactX)
	*lastYZ = (uint32(impactY) << 16) | uint32(impactZ)
	return out, nil
}

func (d *decoder) decodeSVCNails(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	countByte, err := rd.DecodeFreqByte(ft, freq.SVCNailsProjectileCount)
	if err != nil {
		return nil, err
	}
	out = append(out, countByte)

	b0 := byte(uint16(d.primaryPlayerPosXY) >> 4)
	b1 := (byte(d.primaryPlayerPosXY>>16) & 0xf0) | (byte(d.primaryPlayerPosXY>>8) >> 4)
	b2 := byte(d.primaryPlayerPosXY >> 24)
	b3 := byte(uint16(d.primaryPlayerPosZ) >> 4)
	b4 := byte(int8(byte(d.primaryPlayerPosZ>>8)) >> 4)
	b5 := byte(0)

	count := int(countByte)
	for i := 0; i < count; i++ {
		v, err := rd.DecodeFreqByte(ft, freq.SVCNailsProjectileByte0)
		if err != nil {
			return nil, err
		}
		b0 = byte(int(b0) + int(int8(v)))
		out = append(out, b0)

		v, err = rd.DecodeFreqByte(ft, freq.SVCNailsProjectileByte1)
		if err != nil {
			return nil, err
		}
		b1 = byte(int(b1) + int(int8(v)))
		out = append(out, b1)

		v, err = rd.DecodeFreqByte(ft, freq.SVCNailsProjectileByte2)
		if err != nil {
			return nil, err
		}
		b2 = byte(int(b2) + int(int8(v)))
		out = append(out, b2)

		v, err = rd.DecodeFreqByte(ft, freq.SVCNailsProjectileByte3)
		if err != nil {
			return nil, err
		}
		b3 = byte(int(b3) + int(int8(v)))
		out = append(out, b3)

		v, err = rd.DecodeFreqByte(ft, freq.SVCNailsProjectileByte4)
		if err != nil {
			return nil, err
		}
		b4 = byte(int(b4) + int(int8(v)))
		out = append(out, b4)

		v, err = rd.DecodeFreqByte(ft, freq.SVCNailsProjectileByte5)
		if err != nil {
			return nil, err
		}
		b5 = byte(int(b5) + int(int8(v)))
		out = append(out, b5)
	}

	return out, nil
}
