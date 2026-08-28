package assets

// Qizmo builds its client-string dictionaries from ranges in the service
// string tables.
const (
	clientProxyStringStart  = 293
	clientProxyStringCount  = 27
	clientCommonStringStart = 320
	clientCommonStringCount = 192
)

var (
	ClientProxyStrings  = CenterPrintStrings[clientProxyStringStart : clientProxyStringStart+clientProxyStringCount]
	ClientCommonStrings = PrintChatStrings[clientCommonStringStart : clientCommonStringStart+clientCommonStringCount]
)
