package ui

func TextField(label string, maxLength int, get func(*Session) string, set func(*Session, string) error) Item {
	return Item{
		Label: label,
		Value: get,
		Select: func(s *Session) error {
			return s.Prompt(Prompt{Title: label, MaxLength: maxLength, Submit: set})
		},
	}
}
