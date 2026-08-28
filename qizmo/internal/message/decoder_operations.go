package message

import (
	"fmt"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
)

func (d *packetDecoder) decodeTrailingOperation(
	out []byte,
	svcCode byte,
	options DecodingOptions,
) ([]byte, error) {
	if codec, ok := fixedOperationCodecs[svcCode]; ok {
		return d.decodeFixedOperation(out, codec)
	}

	switch svcCode {
	case protocol.SVCSound:
		return d.decodeSVCSound(out)
	case protocol.SVCPrint:
		return d.decodeSVCPrint(out)
	case protocol.SVCStuffText:
		return d.decodeSVCStuffText(out)
	case protocol.SVCDamage:
		return d.decodeSVCDamage(out)
	case protocol.SVCCenterPrint:
		return d.decodeSVCCenterPrint(out)
	case protocol.SVCTempEntity:
		return d.decodeSVCTempEntity(out)
	case protocol.SVCMuzzleFlash:
		return d.decodeSVCMuzzleFlash(out)
	case protocol.SVCUpdateUserInfo:
		return d.decodeSVCUpdateUserInfo(out)
	case protocol.SVCLightStyle:
		return d.decodeSVCLightStyle(out)
	case protocol.SVCServerInfo:
		return d.decodeSVCServerInfo(out)
	case protocol.SVCSetInfo:
		return d.decodeSVCSetInfo(out)
	case protocol.SVCNails:
		return d.decodeSVCNails(out)
	case protocol.SVCUpdatePing:
		return d.decodeSVCUpdatePing(out)
	case protocol.SVCUpdatePL:
		return d.decodeSVCUpdatePL(out)
	case qizmoprotocol.SVCString:
		return d.decodeString(out, freq.ByteValue)
	case qizmoprotocol.SVCVoice:
		return d.decodeSVCVoice(out)
	case protocol.SVCPlayerInfo:
		return d.decodeSVCPlayerInfoDeltas(out)
	case protocol.SVCPacketEntities:
		body, err := d.decodeSVCPacketEntities(false)
		return append(out, body...), err
	case protocol.SVCDeltaPacketEntities:
		body, err := d.decodeSVCPacketEntities(options.PreservePacketEntityDeltas)
		if err != nil {
			return nil, err
		}
		_, hasPacketBase := d.state.PacketBase()
		if !options.PreservePacketEntityDeltas || !hasPacketBase {
			out[len(out)-1] = protocol.SVCPacketEntities
		}
		return append(out, body...), nil
	default:
		return nil, fmt.Errorf("unsupported svc opcode 0x%02x", svcCode)
	}
}

func (d *packetDecoder) decodeSVCMuzzleFlash(out []byte) ([]byte, error) {
	lo, err := d.rd.DecodeFreqByte(d.ft, freq.SVCTEntBeamEntityLo)
	if err != nil {
		return nil, err
	}
	hi, err := d.rd.DecodeFreqByte(d.ft, freq.SVCTEntBeamEntityHi)
	if err != nil {
		return nil, err
	}

	entity := d.lastEntity ^ uint16(lo) ^ uint16(hi)<<8
	d.lastEntity = entity
	return appendUint16LE(out, entity), nil
}

func (d *packetDecoder) decodeFixedOperation(
	out []byte,
	codec fixedOperationCodec,
) ([]byte, error) {
	for i := 0; i < codec.size; i++ {
		value, err := d.rd.DecodeFreqByte(d.ft, codec.row(i))
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func (d *packetDecoder) appendFreqBytes(
	out []byte,
	freqTableAddr uint32,
	count int,
) ([]byte, error) {
	for range count {
		b, err := d.rd.DecodeFreqByte(d.ft, freqTableAddr)
		if err != nil {
			return nil, err
		}

		out = append(out, b)
	}

	return out, nil
}
