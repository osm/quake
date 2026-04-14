package bsp

import (
	"bytes"
	"os"
	"testing"
)

func TestBSP(t *testing.T) {
	fixtures := []string{
		"testdata/e3m2tdm.bsp",
		"testdata/ultrav.bsp",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			data, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			bsp, err := Parse(data)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			if !bytes.Equal(data, bsp.Bytes()) {
				t.Fatalf("serialized BSP does not match original")
			}
		})
	}
}

func TestHasLineOfSight(t *testing.T) {
	bsp := &BSP{
		Planes: []Plane{
			{
				Normal: [3]float32{1, 0, 0},
				Dist:   0,
			},
		},
		Nodes: []Node{
			{
				Plane:    0,
				Children: [2]int16{-1, -2},
			},
		},
		Leafs: []Leaf{
			{Contents: contentsEmpty},
			{Contents: contentsSolid},
		},
		Models: []Model{
			{
				Headnodes: [4]int32{0, 0, 0, 0},
			},
		},
	}

	if !bsp.HasLineOfSight(
		[3]float64{8, 0, 0},
		[3]float64{4, 0, 0},
	) {
		t.Fatalf("expected clear line of sight through empty leaf")
	}

	if bsp.HasLineOfSight(
		[3]float64{8, 0, 0},
		[3]float64{-8, 0, 0},
	) {
		t.Fatalf("expected blocked line of sight through solid leaf")
	}
}
