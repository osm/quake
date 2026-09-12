package ui

import (
	"fmt"
	"math"
	"strings"
)

// Synchronize background progress updates with get to avoid data races.
func Progress(label string, get func(*Session) float64) Item {
	return Item{Label: label, progress: get}
}

func renderProgress(value float64, width int, style Style) string {
	if math.IsNaN(value) {
		value = 0
	}
	value = max(0, min(value, 1))
	cells := width - 7
	track := []byte(strings.Repeat(string([]byte{style.ProgressTrack}), cells))
	track[int(value*float64(cells-1))] = style.ProgressThumb
	bar := string([]byte{style.ProgressLeft}) + string(track) + string([]byte{style.ProgressRight})
	return bar + fmt.Sprintf(" %3d%%", int(value*100))
}
