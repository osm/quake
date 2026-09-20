package mvz

import "github.com/osm/quake/demo/mvz/internal/codec"

// FormatVersion is the MVZ version produced by Encode.
const FormatVersion uint16 = codec.FormatVersion

// Encode compresses an MVD demo in chunks aligned to records, targeting 512 KiB.
// It runs in the calling goroutine and returns the complete MVZ file.
func Encode(mvdData []byte) ([]byte, error) {
	return codec.Encode(mvdData)
}

// EncodeWithWorkers is like Encode but uses up to workers concurrent workers
// for profile selection and chunk compression. Parsing stays sequential.
// Workers must be at least one. One runs in the calling goroutine.
// The worker count does not affect the encoded bytes.
func EncodeWithWorkers(mvdData []byte, workers int) ([]byte, error) {
	return codec.EncodeWithWorkers(mvdData, workers)
}

// Decode returns the entire original MVD, rejecting bytes after the end entry.
// It runs in the calling goroutine and buffers the complete output in memory.
// The format bounds chunks, not total output.
func Decode(mvzData []byte) ([]byte, error) {
	return codec.Decode(mvzData)
}

// DecodeWithWorkers is like Decode but decodes independent chunks using up to
// workers concurrent workers. Workers must be at least one. One runs in the
// calling goroutine.
func DecodeWithWorkers(mvzData []byte, workers int) ([]byte, error) {
	return codec.DecodeWithWorkers(mvzData, workers)
}
