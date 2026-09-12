package main

import (
	"fmt"
	"strings"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/server"
)

func (c *chat) message(client server.Client, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	p := c.clients[client]
	if p == nil {
		return
	}
	if p.channel == "" {
		p.enqueue(protocol.PrintHigh, "Join a channel through the menu before chatting.")
		return
	}
	c.broadcast(p.channel, protocol.PrintChat, fmt.Sprintf("[%s] %s: %s", p.channel, clientName(client), text))
}

func (c *chat) broadcast(channel string, level byte, text string) {
	for _, p := range c.clients {
		if p.channel == channel {
			p.enqueue(level, text)
		}
	}
}

func chatText(text string) (string, bool) {
	text = strings.TrimSpace(text)
	end := strings.IndexAny(text, " \t")
	if end < 0 {
		end = len(text)
	}
	cmd := strings.ToLower(text[:end])
	if cmd != "say" && cmd != "say_team" {
		return "", false
	}

	value := strings.TrimSpace(text[end:])
	if strings.HasPrefix(value, "\"") {
		if len(value) < 2 || !strings.HasSuffix(value, "\"") {
			return "", true
		}
		value = value[1 : len(value)-1]
	}
	return cleanText(value, 160), true
}

func cleanText(text string, limit int) string {
	b := []byte(text)
	for i, ch := range b {
		if ch < 32 || ch == 127 || ch == 255 {
			b[i] = ' '
		}
	}
	text = strings.TrimSpace(string(b))
	return text[:min(len(text), limit)]
}
