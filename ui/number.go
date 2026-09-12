package ui

import "strconv"

func Number(label string, lower, upper, step int, get func(*Session) int, set func(*Session, int) error) Item {
	return Item{
		Label:    label,
		Value:    func(s *Session) string { return strconv.Itoa(get(s)) },
		Disabled: func(*Session) bool { return lower > upper || step <= 0 },
		Adjust: func(s *Session, direction int) error {
			if lower > upper || step <= 0 || direction == 0 {
				return nil
			}
			value := get(s)
			next := numberStep(max(lower, min(value, upper)), lower, upper, step, direction)
			if next == value {
				return nil
			}
			return set(s, next)
		},
	}
}

func numberStep(value, lower, upper, step, direction int) int {
	if direction < 0 {
		next := value - step
		if next > value || next < lower {
			return lower
		}
		return next
	}

	next := value + step
	if next < value || next > upper {
		return upper
	}
	return next
}
