package oracle

import (
	"encoding/json"
	"fmt"
	"sort"
)

const ManifestVersion = 2

type Result struct {
	Name          string `json:"name"`
	Family        string `json:"family"`
	InputSize     int    `json:"input_size"`
	InputSHA256   string `json:"input_sha256"`
	QWZSize       int    `json:"qwz_size"`
	QWZSHA256     string `json:"qwz_sha256"`
	DecodedSize   int    `json:"decoded_size"`
	DecodedSHA256 string `json:"decoded_sha256"`
}

type Manifest struct {
	Version       int            `json:"version"`
	Oracle        string         `json:"oracle"`
	SeedQWZSHA256 string         `json:"seed_qwz_sha256"`
	BaseQWDSHA256 string         `json:"base_qwd_sha256"`
	Families      map[string]int `json:"families"`
	Results       []Result       `json:"results"`
}

func NewManifest(results []Result) Manifest {
	families := make(map[string]int)
	for _, result := range results {
		families[result.Family]++
	}
	return Manifest{
		Version:       ManifestVersion,
		Oracle:        "QW Qizmo v2.91, fresh -C and -D processes",
		SeedQWZSHA256: SeedQWZSHA256,
		BaseQWDSHA256: BaseQWDSHA256,
		Families:      families,
		Results:       results,
	}
}

func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode oracle manifest: %w", err)
	}
	if manifest.Version != ManifestVersion {
		return Manifest{}, fmt.Errorf("manifest version %d, want %d", manifest.Version, ManifestVersion)
	}
	if manifest.SeedQWZSHA256 != SeedQWZSHA256 || manifest.BaseQWDSHA256 != BaseQWDSHA256 {
		return Manifest{}, fmt.Errorf("manifest seed/base checksums do not match scenario generator")
	}
	if err := validateResults(manifest.Results); err != nil {
		return Manifest{}, err
	}
	families := make(map[string]int)
	for _, result := range manifest.Results {
		families[result.Family]++
	}
	if len(families) != len(manifest.Families) {
		return Manifest{}, fmt.Errorf("manifest family count has %d entries, want %d", len(manifest.Families), len(families))
	}
	for family, count := range families {
		if manifest.Families[family] != count {
			return Manifest{}, fmt.Errorf("manifest family %s count %d, want %d", family, manifest.Families[family], count)
		}
	}
	return manifest, nil
}

func MarshalManifest(manifest Manifest) ([]byte, error) {
	if err := validateResults(manifest.Results); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func validateResults(results []Result) error {
	if len(results) == 0 {
		return fmt.Errorf("manifest has no oracle results")
	}
	previous := ""
	for i, result := range results {
		if result.Name == "" || result.Family == "" || result.InputSize <= 0 || result.QWZSize <= 0 || result.DecodedSize < 0 {
			return fmt.Errorf("oracle result %d has invalid metadata", i)
		}
		if len(result.InputSHA256) != 64 || len(result.QWZSHA256) != 64 || len(result.DecodedSHA256) != 64 {
			return fmt.Errorf("oracle result %s has invalid checksum length", result.Name)
		}
		if previous != "" && result.Name <= previous {
			return fmt.Errorf("oracle results are not uniquely sorted at %q after %q", result.Name, previous)
		}
		previous = result.Name
	}
	return nil
}

func SortResults(results []Result) {
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
}
