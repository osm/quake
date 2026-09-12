package ui

import (
	"fmt"
	"strings"
)

func (s *Session) Confirm(title string, accept func(*Session) error) error {
	if accept == nil {
		return fmt.Errorf("ui: confirmation requires a callback")
	}

	page := &Page{Title: title, confirm: true}
	page.Items = []Item{
		{Label: "Yes", Select: func(s *Session) error {
			depth := len(s.stack)
			if err := accept(s); err != nil {
				return err
			}
			if len(s.stack) == depth && s.Current() == page && s.visible && s.prompt == nil {
				s.stack = s.stack[:depth-1]
			}
			return nil
		}},
		{Label: "No", Select: func(s *Session) error { return s.Handle(Back) }},
	}
	return s.Push(page)
}

func renderButtons(items []Item, selected, width int, style Style) string {
	choices := make([]string, len(items))
	for i, item := range items {
		marker := style.Action
		label := style.ActionColor.text(clean(item.Label))
		if i == selected {
			marker = style.Selected
			label = style.SelectedColor.text(label)
		}
		choices[i] = label
		if len(items) > 1 {
			choices[i] = string([]byte{marker, ' '}) + label
		}
	}
	return style.row(center(strings.Join(choices, " / "), width))
}
