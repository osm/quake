package ui

import (
	"fmt"
	"strings"
)

type Prompt struct {
	Title     string
	MaxLength int
	Submit    func(*Session, string) error
	err       string
	edit      bool
}

func (s *Session) Prompt(prompt Prompt) error {
	if s.root == nil || prompt.Submit == nil {
		return fmt.Errorf("ui: prompt requires a menu and submit callback")
	}
	if prompt.MaxLength == 0 {
		prompt.MaxLength = 64
	}
	if prompt.MaxLength < 1 || prompt.MaxLength > 256 {
		return fmt.Errorf("ui: prompt length must be between 1 and 256 bytes")
	}

	prompt.err, prompt.edit = "", true
	s.prompt = &prompt
	s.visible = true
	return nil
}

func (s *Session) Submit(text string) error {
	p := s.prompt
	if p == nil {
		return fmt.Errorf("ui: no active prompt")
	}
	if len(text) > p.MaxLength {
		return p.reject(fmt.Errorf("Maximum %d bytes", p.MaxLength))
	}
	if strings.ContainsAny(text, "\x00\r\n") || strings.IndexByte(text, 0xff) >= 0 {
		return p.reject(fmt.Errorf("Enter a single line of text"))
	}

	s.prompt = nil
	err := p.Submit(s, text)
	if err != nil && s.prompt == nil && s.visible {
		s.prompt = p
		return p.reject(err)
	}

	return err
}

func (p *Prompt) reject(err error) error {
	p.err = err.Error()
	p.edit = true
	return err
}

func (s *Session) handlePrompt(event Event) {
	switch event {
	case Back:
		s.prompt = nil
	case Select:
		s.prompt.edit = true
	}
}

func (s *Session) renderPrompt() string {
	width, _ := s.dimensions()
	inside := width - 2
	style := s.Style.normalized()
	lines := style.header(s.prompt.Title, inside)
	lines = append(lines,
		style.textRow("Type, then Enter", inside),
		style.textRow("Esc then Backspace", inside),
		style.textRow("to cancel", inside),
	)
	if s.prompt.err != "" {
		lines = append(lines, style.textRow(s.prompt.err, inside))
	}
	lines = append(lines, border(style.BottomLeft, style.Bottom, style.BottomRight, inside))

	return strings.Join(lines, "\n")
}
