package assets

type Assets struct {
	CenterPrintStrings map[uint16][]byte
	PrintStrings       map[uint16][]byte
	PrintChatStrings   map[uint16][]byte
	StuffTextStrings   map[uint16][]byte
	SetInfoStrings     map[uint16][]byte
	PrecacheModels     []string
	PrecacheSounds     []string
}

func stringTable(src []string) map[uint16][]byte {
	dst := make(map[uint16][]byte, len(src))
	for i, value := range src {
		dst[uint16(i)] = []byte(value)
	}
	return dst
}

func Embedded() Assets {
	return Assets{
		CenterPrintStrings: stringTable(CenterPrintStrings),
		PrintStrings:       stringTable(PrintStrings),
		PrintChatStrings:   stringTable(PrintChatStrings),
		StuffTextStrings:   stringTable(StuffTextStrings),
		SetInfoStrings:     stringTable(SetInfoStrings),
		PrecacheModels:     PrecacheModels,
		PrecacheSounds:     PrecacheSounds,
	}
}
