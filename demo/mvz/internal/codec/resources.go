package codec

import (
	"fmt"

	"github.com/osm/quake/demo/mvz/frequency"
)

// RangeResources holds validated frequency profiles. Its zero value is invalid.
type RangeResources struct {
	// Profiles are probed in order on the first chunk. Later chunks use the
	// selected profile, while the decoder follows the profile stored on wire.
	profiles []rangeProfile
}

type rangeProfile struct {
	id     byte
	models *frequency.Models
}

func newResources(
	profiles ...rangeProfile,
) (*RangeResources, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("MVZ range resources have no profiles")
	}
	seen := make(map[byte]bool, len(profiles))
	validated := make([]rangeProfile, len(profiles))
	for i, profile := range profiles {
		if profile.id == 0 {
			return nil, fmt.Errorf("MVZ frequency profile ID 0 is reserved")
		}
		if profile.models == nil {
			return nil, fmt.Errorf("MVZ frequency profile %d has nil models", profile.id)
		}
		if seen[profile.id] {
			return nil, fmt.Errorf("duplicate MVZ frequency profile %d", profile.id)
		}
		seen[profile.id] = true
		if err := profile.models.Validate(); err != nil {
			return nil, fmt.Errorf("validate MVZ frequency profile %d: %w", profile.id, err)
		}
		copyModels := *profile.models
		profile.models = &copyModels
		validated[i] = profile
	}
	return &RangeResources{profiles: validated}, nil
}

func findProfile(profiles []rangeProfile, id byte) (rangeProfile, bool) {
	for _, candidate := range profiles {
		if candidate.id == id {
			return candidate, true
		}
	}
	return rangeProfile{}, false
}
