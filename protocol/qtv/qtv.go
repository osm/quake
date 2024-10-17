package qtv

import "github.com/osm/quake/protocol"

const (
	ProtocolVersion = "QTV_EZQUAKE_EXT"
	Version         = 1
)

const (
	ExtensionDownload = 1 << 0
	ExtensionSetInfo  = 1 << 1
	ExtensionUserList = 1 << 2
)

const (
	CLCStringCmd protocol.CommandType = 1
)
