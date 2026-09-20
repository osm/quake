package codec

import (
	"fmt"
	"sync"

	"github.com/osm/quake/demo/mvz/frequency"
)

var version1Cache struct {
	sync.Once
	resources *RangeResources
	err       error
}

func version1Resources() (*RangeResources, error) {
	version1Cache.Do(func() {
		version1Cache.resources, version1Cache.err = loadVersion1()
	})
	return version1Cache.resources, version1Cache.err
}

func loadVersion1() (*RangeResources, error) {
	frequencySet, err := frequency.SetV1()
	if err != nil {
		return nil, fmt.Errorf("load MVZ frequency set v1: %w", err)
	}
	profiles := make([]rangeProfile, 0, len(frequencySet))
	for _, profile := range frequencySet {
		profiles = append(profiles, rangeProfile{
			id:     profile.ID,
			models: profile.Models,
		})
	}
	return newResources(profiles...)
}

func resourcesForVersion(version uint16) (*RangeResources, error) {
	switch version {
	case FormatVersion:
		return version1Resources()
	default:
		return nil, fmt.Errorf("unsupported MVZ version %d", version)
	}
}
