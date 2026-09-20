package frequency

import (
	"fmt"
	"math"
)

const (
	Symbols         = 256
	cumulativeTotal = math.MaxUint32
)

// Model selects a row of cumulative frequency bounds.
type Model uint16

// Models holds one complete frequency profile. Cumulative[model][symbol] is
// the upper bound of that symbol's range. The preceding bound (or zero for
// the first symbol) is its lower bound.
type Models struct {
	Cumulative [ModelCount][Symbols]uint32
}

// Row returns the cumulative bounds for a model.
func (m *Models) Row(model Model) ([]uint32, error) {
	if m == nil {
		return nil, fmt.Errorf("nil MVZ frequency models")
	}
	if model >= ModelCount {
		return nil, fmt.Errorf("unknown MVZ frequency model %d", model)
	}
	return m.Cumulative[model][:], nil
}

// Validate checks that every symbol has an interval of positive size and
// that every row ends at the maximum uint32 value.
func (m *Models) Validate() error {
	if m == nil {
		return fmt.Errorf("nil MVZ frequency models")
	}
	for model := Model(0); model < ModelCount; model++ {
		if err := m.validateRow(model); err != nil {
			return err
		}
	}
	return nil
}

func (m *Models) validateRow(model Model) error {
	var previous uint32
	for symbol, bound := range &m.Cumulative[model] {
		if bound <= previous {
			return fmt.Errorf("MVZ model %d symbol %d has an empty interval", model, symbol)
		}
		previous = bound
	}
	if previous != cumulativeTotal {
		return fmt.Errorf(
			"MVZ model %d total %#x, want %#x",
			model, previous, uint32(cumulativeTotal),
		)
	}
	return nil
}
