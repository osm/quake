package qizmo

const (
	// UserInfoKey and UserInfoValue are how Qizmo 2.91 identifies another
	// compression-capable proxy during the QuakeWorld connect exchange.
	UserInfoKey   = "Qizmo"
	UserInfoValue = "2.91 notimer"
)

const (
	// These occupy the client-to-server command stream between cooperating
	// proxies; they are not ordinary QuakeWorld CLCs. The Qizmo 2.91 cases are
	// at 0x0805a190 (encode) and 0x0805be3c (decode).
	CLCPeerAssociation  = 0x33
	CLCVoiceStart       = 0x50
	CLCVoiceStop        = 0x51
	CLCVoiceRaw         = 0x52
	CLCVoiceGSM         = 0x53
	CLCRequestS2CResync = 0x55
)

const (
	SVCString = 81
	SVCBlock  = 82
	SVCVoice  = 83
)

const (
	CLCPeerAssociationPayloadSize = 16
	CLCVoiceRawPayloadSize        = 162
	CLCVoiceGSMPayloadSize        = 34
	SVCBlockPayloadSize           = 162
	SVCVoicePayloadSize           = 34
)

// Qizmo's compressed user commands use this private mask layout.
const (
	CMButtons        = 1 << 12
	CMMsec           = 1 << 13
	CMImpulse        = 1 << 14
	CMInvalid        = 1 << 15
	CMPredictionMask = 0x203f
)

func IsClientExtensionOpcode(opcode byte) bool {
	switch opcode {
	case CLCPeerAssociation,
		CLCVoiceStart,
		CLCVoiceStop,
		CLCVoiceRaw,
		CLCVoiceGSM,
		CLCRequestS2CResync:
		return true
	default:
		return false
	}
}

func IsServiceOpcode(opcode byte) bool {
	switch opcode {
	case SVCString, SVCBlock, SVCVoice:
		return true
	default:
		return false
	}
}
