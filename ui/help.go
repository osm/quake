package ui

func (s *Session) help(f *frame) error {
	page := f.page.Help
	if f.selected >= 0 && !f.selectionLost && f.items[f.selected].Help != nil {
		page = f.items[f.selected].Help
	}
	if page == nil || page == f.page {
		return nil
	}
	return s.Push(page)
}
