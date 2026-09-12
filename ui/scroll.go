package ui

func (s *Session) normalizeScroll(f *frame) {
	_, rows := s.dimensions()
	f.top = max(0, min(f.top, len(f.items)-rows))
	end := min(len(f.items), f.top+rows)
	if f.selected < f.top || f.selected >= end || !f.items[f.selected].selectable(s) {
		s.focusScroll(f, f.top, end, 1)
	}
}

func (s *Session) scrollPage(f *frame, event Event) bool {
	_, rows := s.dimensions()
	switch event {
	case Up:
		s.scrollStep(f, -1, rows)
	case Down:
		s.scrollStep(f, 1, rows)
	case Home:
		f.top, f.selected = 0, -1
	case End:
		f.top = max(0, len(f.items)-rows)
		s.focusScroll(f, len(f.items)-1, f.top-1, -1)
	case PageUp:
		f.top -= rows
	case PageDown:
		f.top += rows
	default:
		return false
	}
	s.normalizeScroll(f)
	return true
}

func (s *Session) scrollStep(f *frame, direction, rows int) {
	if f.selected < 0 || !s.seek(f, f.selected+direction, direction) {
		f.top += direction
		return
	}
	f.top = min(f.top, f.selected)
	f.top = max(f.top, f.selected-rows+1)
}

func (s *Session) focusScroll(f *frame, start, end, direction int) {
	f.selected = -1
	for i := start; i != end; i += direction {
		if !f.items[i].selectable(s) {
			continue
		}
		f.selected = i
		return
	}
}
