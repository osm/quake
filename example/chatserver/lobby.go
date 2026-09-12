package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/osm/quake/server/quake"
)

func parseLobby(mapName, originText, anglesText string) (quake.Lobby, error) {
	origin, err := parseVector(originText)
	if err != nil {
		return quake.Lobby{}, fmt.Errorf("invalid -origin: %w", err)
	}
	for _, coordinate := range origin {
		if coordinate < -4096 || coordinate > 4095.875 {
			return quake.Lobby{}, fmt.Errorf("invalid -origin: coordinates must be between -4096 and 4095.875 for the classic QuakeWorld protocol")
		}
	}

	angles, err := parseVector(anglesText)
	if err != nil {
		return quake.Lobby{}, fmt.Errorf("invalid -angles: %w", err)
	}

	return quake.Lobby{
		Map:    mapName,
		Title:  "Quake chat server",
		Origin: origin,
		Angles: angles,
	}, nil
}

func parseVector(text string) ([3]float32, error) {
	var result [3]float32
	fields := strings.Fields(text)
	if len(fields) != 3 {
		return result, fmt.Errorf("expected three space-separated numbers")
	}

	for i, field := range fields {
		value, err := strconv.ParseFloat(field, 32)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return result, fmt.Errorf("%q is not a finite number", field)
		}
		result[i] = float32(value)
	}

	return result, nil
}
