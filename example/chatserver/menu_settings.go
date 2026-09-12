package main

import (
	"strings"

	"github.com/osm/quake/ui"
)

func (m *menu) settings(s *ui.Session) error {
	page := &ui.Page{Title: "Settings", Items: []ui.Item{
		m.titleItem(),
		selectionColor(),
	}}
	page.Items = append(page.Items, menuFooter()...)
	return s.Push(page)
}

func selectionColor() ui.Item {
	item := ui.Choice("Selection color", []string{"Red", "White"},
		func(s *ui.Session) int {
			if s.Style.SelectedColor == ui.ColorWhite {
				return 1
			}
			return 0
		},
		func(s *ui.Session, value int) error {
			colors := []ui.Color{ui.ColorBrown, ui.ColorWhite}
			s.Style.SelectedColor = colors[value]
			s.Style.ActionColor = colors[1-value]
			return nil
		},
	)
	item.Help = menuHelp("Selection color",
		"Choose the color of the selected menu item. Other options use the opposite color so the selection stays easy to find.",
		"Left/Right or Enter: change color",
	)
	return item
}

func (m *menu) setTitle(s *ui.Session, title string) error {
	m.root.Title = strings.TrimSpace(title)
	s.Notify("Menu title saved")
	return nil
}

func (m *menu) titleItem() ui.Item {
	item := ui.TextField("Menu title", 24,
		func(*ui.Session) string { return m.root.Title },
		m.setTitle,
	)
	item.Delete = func(s *ui.Session) error { return m.setTitle(s, "") }
	item.Help = menuHelp("Menu title",
		"Change the title of your main menu. This setting only changes it for you; other players keep their own titles.",
		"Enter: edit title",
		"Delete: clear title",
	)
	return item
}
