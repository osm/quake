package mvd

const ProtocolVersion = ('M' << 0) + ('V' << 8) + ('D' << 16) + ('1' << 24)

const (
	ExtensionFloatCoords       = 1 << 0
	ExtensionHighLagTeleport   = 1 << 1
	ExtensionServerSideWeapon  = 1 << 2
	ExtensionDebugWeapon       = 1 << 3
	ExtensionDebugAntilag      = 1 << 4
	ExtensionHiddenMessages    = 1 << 5
	ExtensionServerSideWeapon2 = 1 << 6
	ExtensionIncludeInMVD      = ExtensionHiddenMessages
)

const (
	DemoMultiple = 3
	DemoSingle   = 4
	DemoStats    = 5
	DemoAll      = 6
)

const (
	CLCWeapon              = 200
	CLCWeaponModePresel    = 1 << 0
	CLCWeaponModeIffiring  = 1 << 1
	CLCWeaponForgetRanking = 1 << 2
	CLCWeaponHideAxe       = 1 << 3
	CLCWeaponHideSg        = 1 << 4
	CLCWeaponResetOnDeath  = 1 << 5
	CLCWeaponSwitching     = 1 << 6
	CLCWeaponFullImpulse   = 1 << 7
)

const (
	HiddenAntilagPosition              = 0
	HiddenUserCommand                  = 1
	HiddenUserCommandWeapon            = 2
	HiddenDemoInfo                     = 3
	HiddenCommentaryTrack              = 4
	HiddenCommentaryData               = 5
	HiddenCommentaryTextSegment        = 6
	HiddenDamangeDone                  = 7
	HiddenUserCommandWeaponServerSide  = 8
	HiddenUserCommandWeaponInstruction = 9
	HiddenPausedDuration               = 10
)
