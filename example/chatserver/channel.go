package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/server"
)

var channelName = regexp.MustCompile(`^[a-z0-9_-]{1,16}$`)

func (c *chat) create(client server.Client, name string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if !channelName.MatchString(name) {
		return fmt.Errorf("Use 1-16 letters, digits, - or _")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.channels[name]; exists {
		return fmt.Errorf("Channel already exists")
	}
	if len(c.channels) >= 32 {
		return fmt.Errorf("Channel limit reached")
	}
	if c.clients[client] == nil {
		return fmt.Errorf("Client disconnected")
	}
	c.channels[name] = struct{}{}
	return c.move(client, name)
}

func (c *chat) join(client server.Client, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.channels[name]; name != "" && !exists {
		return fmt.Errorf("Channel no longer exists")
	}
	return c.move(client, name)
}

func (c *chat) move(client server.Client, name string) error {
	p := c.clients[client]
	if p == nil {
		return fmt.Errorf("Client disconnected")
	}
	if p.channel == name {
		return nil
	}
	previous := p.channel
	if previous != "" {
		c.broadcast(previous, protocol.PrintHigh, fmt.Sprintf("%s left #%s.", clientName(client), previous))
	}
	p.channel = name
	c.removeEmpty(previous)
	if name != "" {
		c.broadcast(name, protocol.PrintHigh, fmt.Sprintf("%s joined #%s.", clientName(client), name))
	}
	return nil
}

func (c *chat) removeEmpty(name string) {
	if name != "lobby" && c.memberCount(name) == 0 {
		delete(c.channels, name)
	}
}

func (c *chat) channel(client server.Client) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p := c.clients[client]; p != nil && p.channel != "" {
		return "#" + p.channel
	}
	return "none"
}

func (c *chat) list() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	names := make([]string, 0, len(c.channels))
	for name := range c.channels {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *chat) members(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.memberCount(name)
}

func (c *chat) users(name string) []server.Client {
	c.mu.Lock()
	defer c.mu.Unlock()

	var clients []server.Client
	for client, p := range c.clients {
		if p.channel == name {
			clients = append(clients, client)
		}
	}
	sort.Slice(clients, func(i, j int) bool {
		return clientName(clients[i]) < clientName(clients[j])
	})
	return clients
}

func (c *chat) memberCount(name string) int {
	count := 0
	for _, p := range c.clients {
		if p.channel == name {
			count++
		}
	}
	return count
}
