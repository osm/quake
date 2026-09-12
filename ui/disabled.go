package ui

func (s *Session) explainDisabled(item Item) error {
	if item.DisabledReason == nil {
		return nil
	}
	reason := item.DisabledReason(s)
	if reason == "" {
		return nil
	}
	page := TextPage(item.Label, reason)
	page.Items = []Item{
		{},
		{Label: "OK", Select: func(s *Session) error {
			return s.Handle(Back)
		}},
	}
	return s.Push(page)
}
