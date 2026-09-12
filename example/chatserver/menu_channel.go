package main

import (
	"fmt"
	"strings"

	"github.com/osm/quake/ui"
)

func (m *menu) currentUsers(s *ui.Session) error {
	return m.users(s, strings.TrimPrefix(m.chat.channel(m.client), "#"))
}

func (m *menu) users(s *ui.Session, name string) error {
	page := ui.SearchableList("#"+name+" users", func(*ui.Session) []ui.Item {
		var items []ui.Item
		for _, user := range m.chat.users(name) {
			items = append(items, ui.Item{
				Key:       "user:" + user.GetAddr(),
				Label:     clientName(user),
				Focusable: true,
			})
		}
		return items
	})
	page.Items = menuFooter()
	return s.Push(page)
}

func (m *menu) browse(s *ui.Session) error {
	page := ui.SearchableList("Channels", func(*ui.Session) []ui.Item {
		var items []ui.Item
		for _, name := range m.chat.list() {
			items = append(items, m.channelItem(name))
		}
		return items
	})
	page.Items = menuFooter()
	return s.Push(page)
}

func (m *menu) channelItem(name string) ui.Item {
	return ui.Item{
		Key:   "channel:" + name,
		Label: "#" + name,
		Value: func(*ui.Session) string { return fmt.Sprintf("%d users", m.chat.members(name)) },
		Select: func(s *ui.Session) error {
			if err := m.join(name); err != nil {
				return err
			}
			if err := s.Handle(ui.Back); err != nil {
				return err
			}
			s.Notify("Joined #" + name)
			return nil
		},
	}
}

func menuFooter() []ui.Item {
	return []ui.Item{
		{Separator: true},
		{Key: "back", Label: "Back", Select: func(s *ui.Session) error { return s.Handle(ui.Back) }},
	}
}
