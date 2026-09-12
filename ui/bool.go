package ui

func Bool(label string, get func(*Session) bool, set func(*Session, bool) error) Item {
	return Item{
		Label: label,
		Value: func(s *Session) string {
			if get(s) {
				return "on"
			}
			return "off"
		},
		Select: func(s *Session) error {
			return set(s, !get(s))
		},
		Adjust: func(s *Session, direction int) error {
			return set(s, direction > 0)
		},
	}
}
