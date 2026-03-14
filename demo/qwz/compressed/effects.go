package compressed

import "github.com/osm/quake/demo/qwz/freq"

func (d *decoder) decodeSVCDamage(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	for _, freqTableAddr := range []uint32{
		freq.SVCDamageArmor,
		freq.SVCDamageBlood,
	} {
		b, err := rd.DecodeFreqByte(ft, freqTableAddr)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	hasCoords, err := rd.DecodeFreqSymbol(ft, freq.SVCDamageHasFrom, 2)
	if err != nil {
		return nil, err
	}
	if hasCoords == 0 {
		return append(out, 0, 0, 0, 0, 0, 0), nil
	}

	coordX := uint16(d.primaryPlayerPosXY & 0xffff)
	coordY := uint16(d.primaryPlayerPosXY >> 16)
	coordZ := uint16(d.primaryPlayerPosZ & 0xffff)

	lo, err := rd.DecodeFreqByte(ft, freq.SVCDamageFromXLo)
	if err != nil {
		return nil, err
	}
	hi, err := rd.DecodeFreqByte(ft, freq.SVCDamageFromXHi)
	if err != nil {
		return nil, err
	}
	coordX = uint16(int16(coordX) + int16(int8(lo)) + int16(uint16(hi)<<8))
	out = appendUint16LE(out, coordX)

	lo, err = rd.DecodeFreqByte(ft, freq.SVCDamageFromYLo)
	if err != nil {
		return nil, err
	}
	hi, err = rd.DecodeFreqByte(ft, freq.SVCDamageFromYHi)
	if err != nil {
		return nil, err
	}
	coordY = uint16(int16(coordY) + int16(int8(lo)) + int16(uint16(hi)<<8))
	out = appendUint16LE(out, coordY)

	lo, err = rd.DecodeFreqByte(ft, freq.SVCDamageFromZLo)
	if err != nil {
		return nil, err
	}
	hi, err = rd.DecodeFreqByte(ft, freq.SVCDamageFromZHi)
	if err != nil {
		return nil, err
	}
	coordZ = uint16(int16(coordZ) + int16(int8(lo)) + int16(uint16(hi)<<8))
	return appendUint16LE(out, coordZ), nil
}

func (d *decoder) decodeSVCQizmoVoice(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft

	freqTableAddr := uint32(freq.SVCQizmoVoiceData)
	for i := 0; i < 0x22; i++ {
		b, err := rd.DecodeFreqByte(ft, freqTableAddr)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
		freqTableAddr += 0x400
	}
	return out, nil
}
