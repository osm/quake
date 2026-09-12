package ui

type frame struct {
	page          *Page
	selected      int
	top           int
	items         []Item
	selectionLost bool
	query         string
}

func (f *frame) refresh(s *Session) {
	key := ""
	if f.selected >= 0 && f.selected < len(f.items) {
		key = f.items[f.selected].Key
	}
	items := f.page.Items
	if f.page.source != nil {
		items = f.page.source(s)
	}
	clear(f.items)
	f.items = append(f.items[:0], items...)
	f.selectionLost = false
	if key == "" {
		return
	}

	for i, item := range f.items {
		if item.Key == key {
			f.selected = i
			f.selectionLost = !item.selectable(s)
			return
		}
	}
	f.selectionLost = true
}
