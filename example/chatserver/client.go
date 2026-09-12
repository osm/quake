package main

import (
	"fmt"

	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/server"
)

type chatClient struct {
	channel  string
	messages []command.Command
}

func (c *chat) connect(client server.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.clients[client] != nil {
		return
	}
	p := &chatClient{channel: "lobby"}
	c.clients[client] = p
	p.enqueue(protocol.PrintHigh, "Welcome! Use normal chat to talk and the menu to choose a channel.")
	c.broadcast("lobby", protocol.PrintHigh, fmt.Sprintf("%s joined #lobby.", clientName(client)))
}

func (c *chat) disconnect(client server.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	p := c.clients[client]
	if p == nil {
		return
	}
	delete(c.clients, client)
	if p.channel != "" {
		c.broadcast(p.channel, protocol.PrintHigh, fmt.Sprintf("%s disconnected.", clientName(client)))
		c.removeEmpty(p.channel)
	}
}

func (c *chat) notice(client server.Client, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if p := c.clients[client]; p != nil {
		p.enqueue(protocol.PrintHigh, text)
	}
}

func (c *chat) drain(client server.Client) []command.Command {
	c.mu.Lock()
	defer c.mu.Unlock()

	p := c.clients[client]
	if p == nil {
		return nil
	}
	n := min(2, len(p.messages))
	messages := append([]command.Command(nil), p.messages[:n]...)
	p.messages = p.messages[n:]
	return messages
}

func (p *chatClient) enqueue(level byte, text string) {
	cmd := &print.Command{ID: level, String: text + "\n"}
	if len(p.messages) == 32 {
		copy(p.messages, p.messages[1:])
		p.messages[len(p.messages)-1] = cmd
		return
	}
	p.messages = append(p.messages, cmd)
}

func clientName(client server.Client) string {
	name := cleanText(client.GetName(), 32)
	if name == "" {
		return "unnamed"
	}
	return name
}
