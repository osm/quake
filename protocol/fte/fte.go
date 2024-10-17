package fte

const (
	ProtocolVersion  = ('F' << 0) + ('T' << 8) + ('E' << 16) + ('X' << 24)
	ProtocolVersion2 = ('F' << 0) + ('T' << 8) + ('E' << 16) + ('2' << 24)
)

const (
	ExtensionSetView           = 0x00000001
	ExtensionScale             = 0x00000002
	ExtensionLightStyleCol     = 0x00000004
	ExtensionTrans             = 0x00000008
	ExtensionView2             = 0x00000010
	ExtensionBulletEns         = 0x00000020
	ExtensionAccurateTimings   = 0x00000040
	ExtensionSoundDbl          = 0x00000080
	ExtensionFatness           = 0x00000100
	ExtensionHLBSP             = 0x00000200
	ExtensionTEBullet          = 0x00000400
	ExtensionHullSize          = 0x00000800
	ExtensionModelDbl          = 0x00001000
	ExtensionEntityDbl         = 0x00002000
	ExtensionEntityDbl2        = 0x00004000
	ExtensionFloatCoords       = 0x00008000
	ExtensionVWeap             = 0x00010000
	ExtensionQ2BSP             = 0x00020000
	ExtensionQ3BSP             = 0x00040000
	ExtensionColorMod          = 0x00080000
	ExtensionSplitScreen       = 0x00100000
	ExtensionHexen2            = 0x00200000
	ExtensionSpawnStatic2      = 0x00400000
	ExtensionCustomTempEffects = 0x00800000
	Extension256PacketEntities = 0x01000000
	ExtensionNeverUsed1        = 0x02000000
	ExtensionShowPic           = 0x04000000
	ExtensionSetAttachment     = 0x08000000
	ExtensionNeverUsed2        = 0x10000000
	ExtensionChunkedDownloads  = 0x20000000
	ExtensionCSQC              = 0x40000000
	ExtensionDPFlags           = 0x80000000

	Extension2PrydonCursor      = 0x00000001
	Extension2VoiceChat         = 0x00000002
	Extension2SetAngleDelta     = 0x00000004
	Extension2ReplacementDeltas = 0x00000008
	Extension2MaxPlayers        = 0x00000010
	Extension2PredictionInfo    = 0x00000020
	Extension2NewSizeEncoding   = 0x00000040
	Extension2InfoBlobs         = 0x00000080
	Extension2StunAware         = 0x00000100
	Extension2VRInputs          = 0x00000200
	Extension2LerpTime          = 0x00000400
)

const (
	SVCSpawnStatic    = 21
	SVCModelListShort = 60
	SVCSpawnBaseline  = 66
	SVCVoiceChat      = 84
)

const CLCVoiceChat = 83

const (
	UEvenMore   = 1 << 7
	UScale      = 1 << 0
	UTrans      = 1 << 1
	UFatness    = 1 << 2
	UModelDbl   = 1 << 3
	UUnused1    = 1 << 4
	UEntityDbl  = 1 << 5
	UEntityDbl2 = 1 << 6
	UYetMore    = 1 << 7
	UDrawFlags  = 1 << 8
	UAbsLight   = 1 << 9
	UColourMod  = 1 << 10
	UDPFlags    = 1 << 11
	UTagInfo    = 1 << 12
	ULight      = 1 << 13
	UEffects16  = 1 << 14
	UFarMore    = 1 << 15
)
