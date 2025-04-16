package protocol

const (
	VersionNQ    uint32 = 15
	VersionQW210 uint32 = 25
	VersionQW221 uint32 = 26
	VersionQW    uint32 = 28
)

type CommandType int

const (
	S2CChallenge     = 'c'
	S2CConnection    = 'j'
	A2APing          = 'k'
	A2AAck           = 'l'
	A2ANack          = 'm'
	A2AEcho          = 'e'
	A2CPrint         = 'n'
	S2MHeartbeat     = 'a'
	A2CClientCommand = 'B'
	S2MShutdown      = 'C'
)

const (
	SVCBad                 = 0
	SVCNOP                 = 1
	SVCDisconnect          = 2
	SVCUpdateStat          = 3
	SVCVersion             = 4
	SVCSetView             = 5
	SVCSound               = 6
	SVCTime                = 7
	SVCPrint               = 8
	SVCStuffText           = 9
	SVCSetAngle            = 10
	SVCServerData          = 11
	SVCLightStyle          = 12
	SVCUpdateName          = 13
	SVCUpdateFrags         = 14
	SVCClientData          = 15
	SVCStopSound           = 16
	SVCUpdateColors        = 17
	SVCParticle            = 18
	SVCDamage              = 19
	SVCSpawnStatic         = 20
	SVCSpawnBaseline       = 22
	SVCTempEntity          = 23
	SVCSetPause            = 24
	SVCSignOnNum           = 25
	SVCCenterPrint         = 26
	SVCKilledMonster       = 27
	SVCFoundSecret         = 28
	SVCSpawnStaticSound    = 29
	SVCIntermission        = 30
	SVCFinale              = 31
	SVCCDTrack             = 32
	SVCSellScreen          = 33
	SVCSmallKick           = 34
	SVCBigKick             = 35
	SVCUpdatePing          = 36
	SVCUpdateEnterTime     = 37
	SVCUpdateStatLong      = 38
	SVCMuzzleFlash         = 39
	SVCUpdateUserInfo      = 40
	SVCDownload            = 41
	SVCPlayerInfo          = 42
	SVCNails               = 43
	SVCChokeCount          = 44
	SVCModelList           = 45
	SVCSoundList           = 46
	SVCPacketEntities      = 47
	SVCDeltaPacketEntities = 48
	SVCMaxSpeed            = 49
	SVCEntGravity          = 50
	SVCSetInfo             = 51
	SVCServerInfo          = 52
	SVCUpdatePL            = 53
	SVCNails2              = 54
	SVCQizmoVoice          = 83
)

const (
	CLCBad        = 0
	CLCNOP        = 1
	CLCDoubleMove = 2
	CLCMove       = 3
	CLCStringCmd  = 4
	CLCDelta      = 5
	CLCTMove      = 6
	CLCUpload     = 7
)

const (
	PFMsec        = 1 << 0
	PFCommand     = 1 << 1
	PFVelocity1   = 1 << 2
	PFVelocity2   = 1 << 3
	PFVelocity3   = 1 << 4
	PFModel       = 1 << 5
	PFSkinNum     = 1 << 6
	PFEffects     = 1 << 7
	PFWeaponFrame = 1 << 8
	PFDead        = 1 << 9
	PFGib         = 1 << 10
	PFNoGrav      = 1 << 11
	PFPMCShift    = 11
	PFPMCMask     = 7
	PFOnground    = 1 << 14
	PFSolid       = 1 << 15
)

const (
	PMCNormal         = 0
	PMCNormalJumpHeld = 1
	PMCOldSpectator   = 2
	PMCSpectator      = 3
	PMCFly            = 4
	PMCNone           = 5
	PMCLock           = 6
	PMCExtra3         = 7
)

const (
	CMAngle1  = 1 << 0
	CMAngle3  = 1 << 1
	CMForward = 1 << 2
	CMSide    = 1 << 3
	CMUp      = 1 << 4
	CMButtons = 1 << 5
	CMImpulse = 1 << 6
	CMAngle2  = 1 << 7
	CMMsec    = 1 << 7
)

const (
	DFOrigin      = 1
	DFAngles      = 1 << 3
	DFEffects     = 1 << 6
	DFSkinNum     = 1 << 7
	DFDead        = 1 << 8
	DFGib         = 1 << 9
	DFWeaponFrame = 1 << 10
	DFModel       = 1 << 11
)

const (
	UOrigin1  = 1 << 9
	UOrigin2  = 1 << 10
	UOrigin3  = 1 << 11
	UAngle2   = 1 << 12
	UFrame    = 1 << 13
	URemove   = 1 << 14
	UMoreBits = 1 << 15
)

const (
	UAngle1        = 1 << 0
	UAngle3        = 1 << 1
	UModel         = 1 << 2
	UColorMap      = 1 << 3
	USkin          = 1 << 4
	UEffects       = 1 << 5
	USolid         = 1 << 6
	UCheckMoreBits = (1 << 9) - 1
)

const (
	NQUMoreBits   uint16 = 1 << 0
	NQUOrigin1    uint16 = 1 << 1
	NQUOrigin2    uint16 = 1 << 2
	NQUOrigin3    uint16 = 1 << 3
	NQUAngle2     uint16 = 1 << 4
	NQUNoLerp     uint16 = 1 << 5
	NQUFrame      uint16 = 1 << 6
	NQUSignal     uint16 = 1 << 7
	NQUAngle1     uint16 = 1 << 8
	NQUAngle3     uint16 = 1 << 9
	NQUModel      uint16 = 1 << 10
	NQUColorMap   uint16 = 1 << 11
	NQUSkin       uint16 = 1 << 12
	NQUEffects    uint16 = 1 << 13
	NQULongEntity uint16 = 1 << 14
)

const (
	SoundAttenuation = 1 << 14
	SoundVolume      = 1 << 15

	NQSoundVolume      = 1 << 0
	NQSoundAttenuation = 1 << 1
	NQSoundLooping     = 1 << 2
)

const (
	PrintLow    = 0
	PrintMedium = 1
	PrintHigh   = 2
	PrintChat   = 3
)

const (
	TESpike          = 0
	TESuperSpike     = 1
	TEGunshot        = 2
	TEExplosion      = 3
	TETarExplosion   = 4
	TELightning1     = 5
	TELightning2     = 6
	TEWizSpike       = 7
	TEKnightSpike    = 8
	TELightning3     = 9
	TELavaSplash     = 10
	TETEleport       = 11
	TEBlood          = 12
	TELightningBlood = 13
)

const (
	ButtonAttack  = 1 << 0
	ButtonJump    = 1 << 1
	ButtonUse     = 1 << 2
	ButtonAttack2 = 1 << 3
)

const (
	DemoCmd  = 0
	DemoRead = 1
	DemoSet  = 2
)

const (
	MaxClStats        = 32
	StatHealth        = 0
	StatWeapon        = 2
	StatAmmo          = 3
	StatArmor         = 4
	StatShells        = 6
	StatNails         = 7
	StatRockets       = 8
	StatCells         = 9
	StatActiveWeapon  = 10
	StatTotalSecrets  = 11
	StatTotalMonsters = 12
	StatSecrets       = 13
	StatMonsters      = 14
	StatItems         = 15
)

const (
	ITShotgun         = 1 << 0
	ITSuperShotgun    = 1 << 1
	ITNailgun         = 1 << 2
	ITSuperNailgun    = 1 << 3
	ITGrenadeLauncher = 1 << 4
	ITRocketLauncher  = 1 << 5
	ITLightning       = 1 << 6
	ITSuperLightning  = 1 << 7
	ITShells          = 1 << 8
	ITNails           = 1 << 9
	ITRockets         = 1 << 10
	ITCells           = 1 << 11
	ITAx              = 1 << 12
	ITArmor1          = 1 << 13
	ITArmor2          = 1 << 14
	ITArmor3          = 1 << 15
	ITSuperhealth     = 1 << 16
	ITKey1            = 1 << 17
	ITKey2            = 1 << 18
	ITInvisibility    = 1 << 19
	ITInvulnerability = 1 << 20
	ITSuit            = 1 << 21
	ITQuad            = 1 << 22
	ITSigil1          = 1 << 28
	ITSigil2          = 1 << 29
	ITSigil3          = 1 << 30
	ITSigil4          = 1 << 31
)

const (
	SUViewHeight  = 1 << 0
	SUIdealPitch  = 1 << 1
	SUPunch1      = 1 << 2
	SUPunch2      = 1 << 3
	SUPunch3      = 1 << 4
	SUVelocity1   = 1 << 5
	SUVelocity2   = 1 << 6
	SUVelocity3   = 1 << 7
	SUItems       = 1 << 9
	SUOnGround    = 1 << 10
	SUInWater     = 1 << 11
	SUWeaponFrame = 1 << 12
	SUArmor       = 1 << 13
	SUWeapon      = 1 << 14
)

const DownloadBlockSize = 1024

const (
	PlayerModel = 33168
	EyeModel    = 6967
)

var MapChecksum = map[string]int{
	"dm1":   0xc5c7dab3,
	"dm2":   0x65f63634,
	"dm3":   0x15e20df8,
	"dm4":   0x9c6fe4bf,
	"dm5":   0xb02d48fd,
	"dm6":   0x5208da2b,
	"e1m1":  0xad07d882,
	"e1m2":  0x67100127,
	"e1m3":  0x3546324a,
	"e1m4":  0xedda0675,
	"e1m5":  0xa82c1c8a,
	"e1m6":  0x2c0028e3,
	"e1m7":  0x97d6fb1a,
	"e1m8":  0x4b6e741,
	"e2m1":  0xdcf57032,
	"e2m2":  0xaf961d4d,
	"e2m3":  0xfc992551,
	"e2m4":  0xc3169bc9,
	"e2m5":  0xbf028f3f,
	"e2m6":  0x91a33b81,
	"e2m7":  0x7a3fe018,
	"e3m1":  0x90b20d21,
	"e3m2":  0x9c6c7538,
	"e3m3":  0xc3d05d18,
	"e3m4":  0xb1790cb8,
	"e3m5":  0x917a0631,
	"e3m6":  0x2dc17df8,
	"e3m7":  0x1039c1b1,
	"e4m1":  0xbbf06350,
	"e4m2":  0xfff8cb18,
	"e4m3":  0x59bef08c,
	"e4m4":  0x2d3b183f,
	"e4m5":  0x699ce7f4,
	"e4m6":  0x620ff98,
	"e4m7":  0x9dec01ac,
	"e4m8":  0x3cb46c57,
	"end":   0xbbd4b4a5,
	"start": 0x2a9a3763,
}
