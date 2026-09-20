package codec

import (
	"fmt"
	"math"

	"github.com/osm/quake/demo/mvz/frequency"
)

// TrainingVersion marks files used to evaluate profiles during training.
// Decode rejects these files because they require the exact external resources
// used for encoding.
const TrainingVersion uint16 = 0xffff

// VisitRangeSymbols sends the exact chunking and grammar used by Encode to
// visit. Training tools use this hook to count symbols without duplicating the
// encoder. Returning an error stops the visit.
func VisitRangeSymbols(mvdData []byte, visit func(frequency.Model, byte) error) error {
	if visit == nil {
		return fmt.Errorf("nil MVZ symbol visitor")
	}
	return parseIntoChunks(mvdData, func(chunk parsedRangeChunk) error {
		prepared, err := chunk.prepare()
		if err != nil {
			return err
		}
		return encodeChunkSymbols(
			symbolVisitor(visit), prepared.records, prepared.trailing,
		)
	})
}

type symbolVisitor func(frequency.Model, byte) error

func (visit symbolVisitor) Put(model frequency.Model, symbol byte) error {
	return visit(model, symbol)
}

// EncodeWithResources encodes with external profiles for training.
// Its output requires DecodeWithResources and the same resources.
// It runs in the calling goroutine.
func EncodeWithResources(
	mvdData []byte,
	resources *RangeResources,
) ([]byte, error) {
	if resources == nil {
		return nil, fmt.Errorf("nil MVZ range resources")
	}
	if len(resources.profiles) == 0 {
		return nil, fmt.Errorf("MVZ range resources have no profiles")
	}
	return encodeWithWorkers(
		mvdData, resources, TrainingVersion, 1,
	)
}

// DecodeWithResources decodes training output using the original resources.
// It runs in the calling goroutine.
func DecodeWithResources(
	mvzData []byte,
	resources *RangeResources,
) ([]byte, error) {
	if resources == nil {
		return nil, fmt.Errorf("nil MVZ range resources")
	}
	if len(resources.profiles) == 0 {
		return nil, fmt.Errorf("MVZ range resources have no profiles")
	}
	version, err := parseFileHeader(mvzData)
	if err != nil {
		return nil, err
	}
	if version != TrainingVersion {
		return nil, fmt.Errorf("MVZ version %d is not the training version", version)
	}
	return decodeChunks(mvzData[headerSize:], resources.profiles, 1)
}

// LoadRangeResourcesWithProfiles loads frequency profile artifacts for training.
// IDs are assigned from 1 in argument order, and the first compressed chunk
// selects one for the demo.
func LoadRangeResourcesWithProfiles(
	profileData [][]byte,
) (*RangeResources, error) {
	profiles, err := loadProfiles(profileData)
	if err != nil {
		return nil, err
	}
	return newResources(profiles...)
}

func loadProfiles(profileData [][]byte) ([]rangeProfile, error) {
	if len(profileData) == 0 {
		return nil, fmt.Errorf("no MVZ frequency-profile artifacts")
	}
	if len(profileData) > math.MaxUint8 {
		return nil, fmt.Errorf(
			"too many MVZ frequency-profile artifacts: %d",
			len(profileData),
		)
	}
	profiles := make([]rangeProfile, 0, len(profileData))
	for i, data := range profileData {
		models, err := frequency.ParseModelData(data)
		if err != nil {
			return nil, fmt.Errorf("load MVZ frequency profile %d: %w", i+1, err)
		}
		profiles = append(profiles, rangeProfile{id: byte(i + 1), models: models})
	}
	return profiles, nil
}
