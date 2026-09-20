package codec

const (
	FormatVersion uint16 = 1

	headerSize      = 6
	chunkHeaderSize = 13
	endEntrySize    = 1
	maxChunkSize    = 16 << 20
)

var magic = [4]byte{'M', 'V', 'Z', 0x1a}

const (
	entryTypeEnd byte = iota
	entryTypeData
)
