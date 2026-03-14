package assets

type Assets struct {
	CenterPrintStrings map[uint16][]byte
	PrintStrings       map[uint16][]byte
	PrintMode3Strings  map[uint16][]byte
	StuffTextStrings   map[uint16][]byte
	SetInfoStrings     map[uint16][]byte
	PrecacheModels     []string
	PrecacheSounds     []string
}
