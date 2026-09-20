// Package codec is the shared implementation behind mvz and mvz/tooling.
//
// A chunk combines range-coded record/operation symbols with four DEFLATE
// literal streams. range_encode.go and range_decode.go define the grammar;
// the field codecs predict values and code the differences. Frequency models
// are fixed rows selected from history that both sides reconstruct identically.
package codec

import "fmt"

// Encode targets 512 KiB chunks aligned to records without starting workers.
func Encode(mvdData []byte) ([]byte, error) {
	return EncodeWithWorkers(mvdData, 1)
}

// EncodeWithWorkers limits both profile trials and chunk jobs to workers.
func EncodeWithWorkers(mvdData []byte, workers int) ([]byte, error) {
	if workers < 1 {
		return nil, fmt.Errorf("MVZ worker count must be at least 1, got %d", workers)
	}
	resources, err := resourcesForVersion(FormatVersion)
	if err != nil {
		return nil, err
	}
	return encodeWithWorkers(mvdData, resources, FormatVersion, workers)
}

// Decode returns the entire original MVD without starting workers.
func Decode(mvzData []byte) ([]byte, error) {
	return DecodeWithWorkers(mvzData, 1)
}

// DecodeWithWorkers limits independent chunk jobs to workers.
func DecodeWithWorkers(mvzData []byte, workers int) ([]byte, error) {
	if workers < 1 {
		return nil, fmt.Errorf("MVZ worker count must be at least 1, got %d", workers)
	}
	version, err := parseFileHeader(mvzData)
	if err != nil {
		return nil, err
	}
	resources, err := resourcesForVersion(version)
	if err != nil {
		return nil, err
	}
	return decodeChunks(mvzData[headerSize:], resources.profiles, workers)
}
