package event

import "sort"

const defaultEntityBaseOffset = 8

func (r *Result) Players() []string {
	seen := make(map[string]bool)
	var out []string

	for _, event := range r.Events {
		if event.State == nil {
			continue
		}
		state := event.State
		if state.Player == "" {
			continue
		}
		key := normalizeName(state.Player)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, state.Player)
	}

	sort.Slice(out, func(i, j int) bool {
		return normalizeName(out[i]) < normalizeName(out[j])
	})

	return out
}

func (r *Result) EntityBaseOffset() int {
	if r.maxClients > 0 {
		return r.maxClients
	}

	maxSlot := -1
	for _, event := range r.Events {
		if event.State == nil {
			continue
		}
		state := event.State
		slot := state.Edict - 1
		if slot > maxSlot {
			maxSlot = slot
		}
	}
	if maxSlot >= 0 {
		return maxSlot + 1
	}

	return defaultEntityBaseOffset
}

func (r *Result) MapName() string {
	return r.mapName
}
