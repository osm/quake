package ui

func (s *Session) deleteItem(f *frame) error {
	if f.selected < 0 || f.selectionLost {
		return nil
	}
	item := f.items[f.selected]
	if item.disabled(s) {
		return nil
	}
	if action := item.Delete; action != nil {
		return action(s)
	}
	return nil
}
