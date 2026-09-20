// Package mvz implements the compressed container for multiview QuakeWorld
// MVD demos.
//
// Encode uses range compression for MVD records and preserves their exact bytes.
// It targets 512 KiB chunks aligned to records.
// Encode and Decode run in the calling goroutine. EncodeWithWorkers and
// DecodeWithWorkers opt into bounded parallel work. The worker count must
// be at least one and does not affect the format or encoded bytes. Parsing
// stays sequential, and all four functions buffer complete files.
package mvz
