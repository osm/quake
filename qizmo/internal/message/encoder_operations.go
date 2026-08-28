package message

import (
	"fmt"

	"github.com/osm/quake/protocol"
	qizmoprotocol "github.com/osm/quake/protocol/qizmo"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/rangeenc"
)

func (e *Encoder) encodeOperation(
	enc *rangeenc.Encoder,
	ctx *encodingContext,
	plan operationPlan,
) error {
	encodedOpcode := plan.opcode
	if encodedOpcode == protocol.SVCPacketEntities {
		// Qizmo always uses svc_deltapacketentities in the compressed stream,
		// including packets with back-reference zero. The decoder deliberately
		// reconstructs canonical svc_packetentities bytes.
		encodedOpcode = protocol.SVCDeltaPacketEntities
	}
	if err := enc.EncodeFreqByte(e.ft, freq.SVCType, encodedOpcode); err != nil {
		return err
	}
	data := plan.data
	if codec, ok := fixedOperationCodecs[plan.opcode]; ok {
		return e.encodeFixedOperation(enc, data, codec)
	}
	switch plan.opcode {
	case protocol.SVCSound:
		return e.encodeSVCSound(enc, ctx, data)
	case protocol.SVCPrint:
		return e.encodeSVCPrint(enc, data)
	case protocol.SVCStuffText:
		return e.encodeSVCStuffText(enc, data)
	case protocol.SVCDamage:
		return e.encodeSVCDamage(enc, ctx, data)
	case protocol.SVCTempEntity:
		return e.encodeSVCTempEntity(enc, ctx, data)
	case protocol.SVCCenterPrint:
		return e.encodeSVCCenterPrint(enc, data)
	case qizmoprotocol.SVCVoice:
		return e.encodeSVCVoice(enc, data)
	case protocol.SVCPlayerInfo:
		return e.encodeSVCPlayerInfoDeltas(enc, ctx, plan.playerPlans)
	case protocol.SVCSetInfo:
		return e.encodeSVCSetInfo(enc, data)
	case protocol.SVCLightStyle:
		return e.encodeSVCLightStyle(enc, data)
	case protocol.SVCUpdateUserInfo:
		return e.encodeSVCUpdateUserInfo(enc, data)
	case protocol.SVCServerInfo:
		return e.encodeSVCServerInfo(enc, data)
	case qizmoprotocol.SVCString:
		return e.encodeRepeatedRow(enc, data, freq.ByteValue)
	case protocol.SVCMuzzleFlash:
		return e.encodeSVCMuzzleFlash(enc, ctx, data)
	case protocol.SVCUpdatePing:
		return e.encodeSVCUpdatePing(enc, ctx, data)
	case protocol.SVCUpdatePL:
		return e.encodeSVCUpdatePL(enc, ctx, data)
	case protocol.SVCNails:
		return e.encodeSVCNails(enc, ctx, data)
	case protocol.SVCPacketEntities:
		if err := e.encodeSVCPacketEntities(enc, plan.packetEntities.actions); err != nil {
			return err
		}
		ctx.currentEntities = plan.packetEntities.records
		return nil
	default:
		return fmt.Errorf("unsupported svc opcode 0x%02x", plan.opcode)
	}
}

func (e *Encoder) encodeFixedOperation(
	enc *rangeenc.Encoder,
	data []byte,
	codec fixedOperationCodec,
) error {
	if len(data) != codec.size {
		return fmt.Errorf("payload size %d, want %d", len(data), codec.size)
	}
	for i, value := range data {
		if err := enc.EncodeFreqByte(e.ft, codec.row(i), value); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeRows(enc *rangeenc.Encoder, data []byte, rows ...uint32) error {
	if len(data) != len(rows) {
		return fmt.Errorf("field count %d does not match row count %d", len(data), len(rows))
	}
	for i, b := range data {
		if err := enc.EncodeFreqByte(e.ft, rows[i], b); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeRepeatedRow(enc *rangeenc.Encoder, data []byte, row uint32) error {
	for _, b := range data {
		if err := enc.EncodeFreqByte(e.ft, row, b); err != nil {
			return err
		}
	}
	return nil
}
