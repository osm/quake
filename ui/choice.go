package ui

func Choice(label string, choices []string, get func(*Session) int, set func(*Session, int) error) Item {
	return Item{
		Label: label,
		Value: func(s *Session) string {
			i := get(s)
			if i < 0 || i >= len(choices) {
				return ""
			}
			return choices[i]
		},
		Disabled: func(*Session) bool { return len(choices) == 0 },
		Adjust: func(s *Session, direction int) error {
			if len(choices) == 0 || direction == 0 {
				return nil
			}
			i := get(s)
			if i < 0 || i >= len(choices) {
				i = -1
			}
			if direction < 0 {
				i--
			} else {
				i++
			}
			if i < 0 {
				i = len(choices) - 1
			}
			if i >= len(choices) {
				i = 0
			}
			return set(s, i)
		},
	}
}
