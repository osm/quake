package frequency

import _ "embed"

// Profile identifies one complete set of model frequency tables within a
// frequency set.
type Profile struct {
	ID     byte
	Models *Models
}

const (
	Profile1 byte = 1 + iota
	Profile2
	Profile3
	Profile4
)

// Frequency set v1 contains four immutable profiles. The encoder probes all
// profiles on its first range chunk and records the winning profile ID.

//go:embed v1-profile1.dat
var setV1Profile1Data []byte

//go:embed v1-profile2.dat
var setV1Profile2Data []byte

//go:embed v1-profile3.dat
var setV1Profile3Data []byte

//go:embed v1-profile4.dat
var setV1Profile4Data []byte

// SetV1 loads a fresh copy of every profile in frequency set v1.
func SetV1() ([]Profile, error) {
	embedded := []struct {
		id   byte
		data []byte
	}{
		{Profile1, setV1Profile1Data},
		{Profile2, setV1Profile2Data},
		{Profile3, setV1Profile3Data},
		{Profile4, setV1Profile4Data},
	}
	profiles := make([]Profile, 0, len(embedded))
	for _, item := range embedded {
		models, err := ParseModelData(item.data)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, Profile{ID: item.id, Models: models})
	}
	return profiles, nil
}
