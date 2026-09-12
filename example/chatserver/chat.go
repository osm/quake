package main

import (
	"sync"

	"github.com/osm/quake/packet"
	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/server"
)

type chat struct {
	mu       sync.Mutex
	channels map[string]struct{}
	clients  map[server.Client]*chatClient
}

func newChat() *chat {
	return &chat{
		channels: map[string]struct{}{"lobby": {}},
		clients:  make(map[server.Client]*chatClient),
	}
}

func (c *chat) handle(client server.Client, pkt packet.Packet) server.HandlerResult {
	game, ok := pkt.(*clc.GameData)
	if !ok {
		return server.HandlerResult{}
	}

	out := game.Commands[:0]
	for _, raw := range game.Commands {
		cmd, ok := raw.(*stringcmd.Command)
		if !ok {
			out = append(out, raw)
			continue
		}
		text, matched := chatText(cmd.String)
		if !matched {
			out = append(out, raw)
			continue
		}
		if text != "" {
			c.message(client, text)
		}
	}

	game.Commands = out
	return server.HandlerResult{ClientCommands: c.drain(client)}
}
