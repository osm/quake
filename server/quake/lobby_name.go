package quake

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/protocol"
)

const lobbyNameLength = 31

var lobbyNameCommand = regexp.MustCompile(`^\s*setinfo\s+"?name"?\s+(?:"([^"\r\n]*)"|([^\s"]+))\s*$`)

func (s *Server) lobbyInput(c *client, game *clc.GameData) []command.Command {
	if s.lobby == nil {
		return nil
	}
	var replies []command.Command
	commands := game.Commands[:0]
	for _, raw := range game.Commands {
		cmd, ok := raw.(*stringcmd.Command)
		if !ok {
			commands = append(commands, raw)
			continue
		}
		match := lobbyNameCommand.FindStringSubmatch(cmd.String)
		if match == nil {
			commands = append(commands, raw)
			continue
		}
		name := s.renameLobbyClient(c, match[1]+match[2])
		replies = append(replies, &print.Command{ID: protocol.PrintHigh, String: "Name: " + name + "\n"})
	}
	game.Commands = commands
	return replies
}

func (s *Server) renameLobbyClient(c *client, name string) string {
	players := s.lobbyPlayers()
	players[c.slot] = lobbyPlayer{}
	name = uniqueLobbyName(name, players)
	c.mu.Lock()
	c.name = name
	c.mu.Unlock()
	return name
}

func uniqueLobbyName(name string, players [protocol.QWMaxClients]lobbyPlayer) string {
	base := cleanLobbyName(name)
	used := make(map[string]bool)
	for _, player := range players {
		if player.userID != 0 {
			used[lobbyNameKey(player.name)] = true
		}
	}
	name = base
	for n := 2; used[lobbyNameKey(name)]; n++ {
		prefix := fmt.Sprintf("(%d)", n)
		name = prefix + base[:min(len(base), lobbyNameLength-len(prefix))]
	}
	return name
}

func cleanLobbyName(name string) string {
	var result []byte
	for i := 0; i < len(name) && len(result) < lobbyNameLength; i++ {
		b := name[i]
		if b&127 < 32 || b&127 == 127 || strings.ContainsRune("\\\";", rune(b&127)) {
			continue
		}
		result = append(result, b)
	}
	for len(result) > 0 && result[0]&127 == ' ' {
		result = result[1:]
	}
	for len(result) > 0 && result[len(result)-1]&127 == ' ' {
		result = result[:len(result)-1]
	}
	name = string(result)
	if name == "" {
		return "unnamed"
	}
	return name
}

func lobbyNameKey(name string) string {
	key := []byte(name)
	for i, b := range key {
		b &= 127
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		key[i] = b
	}
	return string(key)
}
