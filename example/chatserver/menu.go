package main

import (
	"github.com/osm/quake/server"
	"github.com/osm/quake/ui"
)

type menu struct {
	chat   *chat
	client server.Client
	root   *ui.Page
}

func (c *chat) menu(client server.Client) *ui.Session {
	c.connect(client)
	m := &menu{chat: c, client: client, root: &ui.Page{Title: "Chat server"}}
	m.refresh()
	return ui.New(m.root)
}

func (m *menu) refresh() {
	m.root.Items = []ui.Item{
		{
			Label: "Channel",
			Value: func(*ui.Session) string { return m.chat.channel(m.client) },
		},
		{Separator: true},
		{Label: "Create channel", Select: m.create},
		{Label: "List channels", Select: m.browse},
		{Label: "Settings", Select: m.settings},
	}
	noChannel := func(*ui.Session) string {
		if m.chat.channel(m.client) == "none" {
			return "Join a channel first."
		}
		return ""
	}
	m.root.Items = append(m.root.Items,
		ui.Item{Separator: true},
		ui.Item{Label: "Channel users", Select: m.currentUsers, DisabledReason: noChannel},
		ui.Item{Label: "Leave channel", Select: m.leave, DisabledReason: noChannel},
	)
}

func (m *menu) create(s *ui.Session) error {
	return s.Prompt(ui.Prompt{
		Title:     "New channel name",
		MaxLength: 16,
		Submit: func(s *ui.Session, name string) error {
			if err := m.chat.create(m.client, name); err != nil {
				return err
			}
			m.refresh()
			s.Notify("Created " + m.chat.channel(m.client))
			return nil
		},
	})
}

func (m *menu) join(name string) error {
	if err := m.chat.join(m.client, name); err != nil {
		m.chat.notice(m.client, err.Error())
		return err
	}
	m.refresh()
	return nil
}

func (m *menu) leave(s *ui.Session) error {
	return s.Confirm("Leave "+m.chat.channel(m.client)+"?", func(*ui.Session) error {
		return m.join("")
	})
}
