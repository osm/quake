package ui

import "fmt"

// Handle reports action errors to the caller and displays them in notification dialogs.
func (s *Session) Handle(event Event) (err error) {
	s.handling++
	defer func() {
		s.handling--
		if err != nil && s.handling == 0 {
			s.Notify(err.Error())
		}
	}()
	return s.handle(event)
}

func (s *Session) handle(event Event) error {
	switch event {
	case Toggle:
		s.visible = !s.visible && s.root != nil
		if !s.visible {
			s.prompt = nil
		}
		return nil
	case Open:
		s.visible = s.root != nil
		return nil
	case Close:
		s.prompt = nil
		s.visible = false
		return nil
	case Up, Down, Left, Right, Select, Delete, Help, Back, Home, End, PageUp, PageDown:
	default:
		return fmt.Errorf("ui: unknown event %q", event)
	}
	if !s.visible {
		return nil
	}
	if s.status != "" {
		s.handleNotification(event)
		return nil
	}

	if s.prompt != nil {
		s.handlePrompt(event)
		return nil
	}

	f := s.normalize()
	if f == nil {
		return nil
	}

	return s.navigate(f, event)
}

func (s *Session) navigate(f *frame, event Event) error {
	if f.page.confirm {
		switch event {
		case Left:
			s.step(f, -1)
			return nil
		case Right:
			s.step(f, 1)
			return nil
		}
	}
	if f.page.scroll && s.scrollPage(f, event) {
		return nil
	}
	switch event {
	case Back:
		if len(s.stack) > 1 {
			s.stack = s.stack[:len(s.stack)-1]
		} else {
			s.visible = false
		}
	case Up:
		s.step(f, -1)
	case Down:
		s.step(f, 1)
	case Home:
		s.seek(f, 0, 1)
	case End:
		s.seek(f, len(f.items)-1, -1)
	case PageUp, PageDown:
		s.pageStep(f, event)
	case Select, Left, Right:
		return s.activate(f, event)
	case Delete:
		return s.deleteItem(f)
	case Help:
		return s.help(f)
	}

	return nil
}

func (s *Session) pageStep(f *frame, event Event) {
	direction := 1
	if event == PageUp {
		direction = -1
	}

	_, rows := s.dimensions()
	target := max(0, min(len(f.items)-1, f.selected+direction*rows))
	if !s.seek(f, target, direction) {
		s.seek(f, target, -direction)
	}
}

func (s *Session) activate(f *frame, event Event) error {
	if f.selected < 0 || f.selectionLost {
		return nil
	}

	item := f.items[f.selected]
	if item.disabled(s) {
		if event == Select {
			return s.explainDisabled(item)
		}
		return nil
	}
	if event != Select && item.Adjust != nil {
		direction := 1
		if event == Left {
			direction = -1
		}
		return item.Adjust(s, direction)
	}
	if event == Select && item.Select != nil {
		return item.Select(s)
	}
	if event == Select && item.Submenu == nil && item.Adjust != nil {
		return item.Adjust(s, 1)
	}
	if event != Left && item.Submenu != nil {
		return s.Push(item.Submenu)
	}

	return nil
}

func (s *Session) step(f *frame, direction int) {
	n := len(f.items)
	if n == 0 {
		return
	}

	for offset := 1; offset <= n; offset++ {
		i := (f.selected + direction*offset + n*2) % n
		if f.items[i].selectable(s) {
			f.selected = i
			return
		}
	}
}

func (s *Session) seek(f *frame, start, direction int) bool {
	for i := start; i >= 0 && i < len(f.items); i += direction {
		if f.items[i].selectable(s) {
			f.selected = i
			return true
		}
	}

	return false
}

func (s *Session) normalize() *frame {
	if len(s.stack) == 0 {
		return nil
	}

	f := &s.stack[len(s.stack)-1]
	f.refresh(s)
	if f.page.scroll {
		s.normalizeScroll(f)
		return f
	}
	if f.selected < 0 || f.selected >= len(f.items) || !f.items[f.selected].selectable(s) {
		start := max(0, min(f.selected, len(f.items)-1))
		f.selected = -1
		if !s.seek(f, start, 1) {
			s.seek(f, start, -1)
		}
	}

	_, rows := s.dimensions()
	if f.selected >= 0 && f.selected < f.top {
		f.top = f.selected
	}
	if f.selected >= 0 && f.selected >= f.top+rows {
		f.top = f.selected - rows + 1
	}
	f.top = max(0, min(f.top, len(f.items)-rows))

	return f
}
