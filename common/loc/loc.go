package loc

import (
	"fmt"
	"strings"
)

type Locations struct {
	locations []Location
}

type Location struct {
	Coord [3]float32
	Name  string
}

func Parse(data []byte) (*Locations, error) {
	var locs []Location

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var x, y, z float64
		var name string
		_, err := fmt.Sscanf(line, "%f %f %f %s", &x, &y, &z, &name)
		if err != nil {
			return nil, fmt.Errorf("error parsing line '%v'", err)
		}

		locs = append(locs, Location{
			Coord: [3]float32{
				float32(x / 8.0),
				float32(y / 8.0),
				float32(z / 8.0),
			},
			Name: trim(name),
		})
	}

	return &Locations{locs}, nil
}

func trim(input string) string {
	m := map[string]string{
		"$loc_name_ga":        "ga",
		"$loc_name_mh":        "mh",
		"$loc_name_pent":      "pent",
		"$loc_name_quad":      "quad",
		"$loc_name_ra":        "ra",
		"$loc_name_ring":      "ring",
		"$loc_name_separator": "-",
		"$loc_name_ya":        "ya",
	}

	for x, y := range m {
		input = strings.Replace(input, x, y, -1)
	}

	return input
}

func (l *Locations) Get(coord [3]float32) *Location {
	var loc *Location
	var min float32

	for _, n := range l.locations {
		x := coord[0] - n.Coord[0]
		y := coord[1] - n.Coord[1]
		z := coord[2] - n.Coord[2]
		d := x*x + y*y + z*z

		if loc == nil || d < min {
			loc = &n
			min = d
		}
	}

	return loc
}
