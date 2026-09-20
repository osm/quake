package tooling

import (
	"github.com/osm/quake/demo/mvz/frequency"
	"github.com/osm/quake/demo/mvz/internal/codec"
)

// TrainingVersion marks files used to evaluate profiles during training.
// Ordinary mvz.Decode rejects these files because they require the exact external
// resources used for encoding.
const TrainingVersion uint16 = codec.TrainingVersion

// RangeResources holds validated frequency profiles for training. Its zero value
// is invalid. Create it with LoadRangeResources or LoadRangeResourcesWithProfiles.
type RangeResources = codec.RangeResources

// VisitRangeSymbols visits model/symbol pairs for training using the encoder's
// grammar and chunk boundaries. Returning an error stops the visit.
func VisitRangeSymbols(mvdData []byte, visit func(frequency.Model, byte) error) error {
	return codec.VisitRangeSymbols(mvdData, visit)
}

// LoadRangeResources loads one frequency profile artifact for training.
// It copies the models so callers cannot mutate the resources.
func LoadRangeResources(profileData []byte) (*RangeResources, error) {
	return LoadRangeResourcesWithProfiles([][]byte{profileData})
}

// LoadRangeResourcesWithProfiles loads frequency profile artifacts for training.
// IDs are assigned from 1 in argument order, and the first compressed chunk
// selects one for the demo.
func LoadRangeResourcesWithProfiles(
	profileData [][]byte,
) (*RangeResources, error) {
	return codec.LoadRangeResourcesWithProfiles(profileData)
}

// EncodeWithResources encodes with external profiles for training.
// Its output requires DecodeWithResources and the same resources.
// It runs in the calling goroutine.
func EncodeWithResources(
	mvdData []byte,
	resources *RangeResources,
) ([]byte, error) {
	return codec.EncodeWithResources(mvdData, resources)
}

// DecodeWithResources decodes training output using the original resources.
// It runs in the calling goroutine.
func DecodeWithResources(
	mvzData []byte,
	resources *RangeResources,
) ([]byte, error) {
	return codec.DecodeWithResources(mvzData, resources)
}
