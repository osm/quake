package ui

import "fmt"

// Sessions hold mutable per-client state; serialize access when used directly.
type Session struct {
	Width     int
	Rows      int
	Style     Style
	Data      any
	root      *Page
	stack     []frame
	prompt    *Prompt
	visible   bool
	status    string
	statusTop int
	handling  int
}

func New(root *Page) *Session {
	s := &Session{root: root}
	s.Reset()

	return s
}

func (s *Session) Reset() {
	s.visible = false
	s.status = ""
	s.statusTop = 0
	s.prompt = nil
	s.stack = nil
	if s.root != nil {
		s.stack = []frame{{page: s.root, selected: -1}}
	}
}

func (s *Session) Visible() bool {
	return s.visible
}

func (s *Session) Current() *Page {
	if len(s.stack) == 0 {
		return nil
	}

	return s.stack[len(s.stack)-1].page
}

func (s *Session) Selected() int {
	f := s.normalize()
	if f == nil {
		return -1
	}

	return f.selected
}

func (s *Session) dimensions() (int, int) {
	width, rows := s.Width, s.Rows
	if width == 0 {
		width = 38
	}
	if rows == 0 {
		rows = 8
	}

	height := s.Style.lineHeight()
	return max(20, min(width, 40)), max(1, min(rows, (12+height-1)/height))
}

func (s *Session) Push(page *Page) error {
	if page == nil || s.root == nil {
		return fmt.Errorf("ui: push requires a page and root menu")
	}

	s.prompt = nil
	s.visible = true
	s.stack = append(s.stack, frame{page: page, selected: -1})
	return nil
}
