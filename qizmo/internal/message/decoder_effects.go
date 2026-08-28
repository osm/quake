package message

import (
	"encoding/binary"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
)

func (d *packetDecoder) decodeSVCDamage(out []byte) ([]byte, error) {
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

	coordinates := d.primaryCoordinates
	if err := decodeCoordinateTriplet(rd, ft, 0x3f, damageCoordinateDeltaRows, &coordinates); err != nil {
		return nil, err
	}
	for _, coordinate := range coordinates {
		out = appendUint16LE(out, coordinate)
	}
	return out, nil
}

func (d *packetDecoder) decodeSVCVoice(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft

	freqTableAddr := uint32(freq.SVCVoiceData)
	for i := 0; i < qizmoprotocol.SVCVoicePayloadSize; i++ {
		b, err := rd.DecodeFreqByte(ft, freqTableAddr)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
		freqTableAddr += 0x400
	}
	return out, nil
}

func (d *packetDecoder) decodeSVCSound(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	basePlayers := d.basePlayers

	channelLo, err := rd.DecodeFreqByte(ft, freq.SVCSoundChannelLo)
	if err != nil {
		return nil, err
	}
	channelHi, err := rd.DecodeFreqByte(ft, freq.SVCSoundChannelHi)
	if err != nil {
		return nil, err
	}
	channelXOR := uint16(channelLo) | uint16(channelHi)<<8
	channel := d.lastEntity<<soundEntityShift ^ channelXOR
	soundPacket := binary.LittleEndian.AppendUint16(nil, channel)
	entity := channel >> soundEntityShift & soundEntityMask
	d.lastEntity = entity
	if channel&protocol.SoundVolume != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCSoundVolume)
		if err != nil {
			return nil, err
		}
		soundPacket = append(soundPacket, b)
	}
	if channel&protocol.SoundAttenuation != 0 {
		b, err := rd.DecodeFreqByte(ft, freq.SVCSoundAttenuation)
		if err != nil {
			return nil, err
		}
		soundPacket = append(soundPacket, b)
	}
	remapIndex, err := rd.DecodeFreqByte(ft, freq.SVCSoundIndex)
	if err != nil {
		return nil, err
	}
	soundIndex := st.SoundForRemapIndex(remapIndex)
	soundPacket = append(soundPacket, soundIndex)
	deltaFlags, err := rd.DecodeFreqByte(ft, freq.SVCSoundFlags)
	if err != nil {
		return nil, err
	}

	coordinates := d.lastCoordinates

	if int8(deltaFlags) >= 0 && entity != 0 {
		if entity < 0x21 {
			if len(basePlayers) != 0 {
				list := st.CurrentPlayers
				if deltaFlags&0x40 != 0 {
					list = basePlayers
				}
				if resolved, ok := playerCoordinates(list, entity); ok {
					coordinates = resolved
				}
			}
		} else {
			if entities, ok := st.EntitySnapshot(st.Sequence()); ok {
				if resolved, ok := soundEntityCoordinates(st, entities, entity); ok {
					coordinates = resolved
				}
			}
		}
	}

	if err := decodeCoordinateTriplet(rd, ft, uint16(deltaFlags), coordinateDeltaRows, &coordinates); err != nil {
		return nil, err
	}
	for _, coordinate := range coordinates {
		soundPacket = appendUint16LE(soundPacket, coordinate)
	}
	d.lastCoordinates = coordinates
	return append(out, soundPacket...), nil
}

func (d *packetDecoder) decodeSVCTempEntity(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	basePlayers := d.basePlayers

	tempEntityType, err := rd.DecodeFreqByte(ft, freq.SVCTEntType)
	if err != nil {
		return nil, err
	}
	out = append(out, tempEntityType)

	impact := d.lastCoordinates

	switch tempEntityShapeForType(tempEntityType) {
	case tempEntityCountedPoint:
		out, err = d.appendFreqBytes(out, freq.SVCTEntCount, 1)
		if err != nil {
			return nil, err
		}
	case tempEntityBeam:
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
			d.lastEntity ^= uint16(lo) ^ uint16(hi)<<8
		}
		entity := d.lastEntity
		out = appendUint16LE(out, entity)
		if int8(flags) >= 0 && entity != 0 {
			if entity < 0x21 {
				if len(basePlayers) != 0 {
					if resolved, ok := playerCoordinates(st.CurrentPlayers, entity); ok {
						impact = resolved
					}
				}
			} else {
				resolved, found, _ := entityCoordinates(
					d.baseEntities,
					entity,
				)
				if found {
					impact = resolved
				}
			}
		}
		if err := decodeCoordinateTriplet(rd, ft, uint16(flags), coordinateDeltaRows, &impact); err != nil {
			return nil, err
		}
		for _, coordinate := range impact {
			out = appendUint16LE(out, coordinate)
		}
	}

	if err := decodeCoordinateTriplet(rd, ft, 0x3f, tempEntityCoordinateDeltaRows, &impact); err != nil {
		return nil, err
	}
	for _, coordinate := range impact {
		out = appendUint16LE(out, coordinate)
	}

	d.lastCoordinates = impact
	return out, nil
}

func (d *packetDecoder) decodeSVCNails(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	countByte, err := rd.DecodeFreqByte(ft, freq.SVCNailsProjectileCount)
	if err != nil {
		return nil, err
	}
	out = append(out, countByte)

	previous := nailProjectileBase(d.primaryCoordinates)

	count := int(countByte)
	for range count {
		for field, row := range nailProjectileDeltaRows {
			delta, err := rd.DecodeFreqByte(ft, row)
			if err != nil {
				return nil, err
			}
			previous[field] += delta
			out = append(out, previous[field])
		}
	}

	return out, nil
}
