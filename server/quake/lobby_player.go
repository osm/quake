package quake

import (
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/updateuserinfo"
	"github.com/osm/quake/protocol"
)

type lobbyPlayer struct {
	name   string
	userID uint32
}

func (s *Server) admitLobbyClient(c *client, name string) bool {
	players := s.lobbyPlayers()
	for slot, player := range players {
		if player.userID != 0 {
			continue
		}
		s.nextUserID++
		c.slot = byte(slot)
		c.userID = s.nextUserID
		c.name = uniqueLobbyName(name, players)
		return true
	}
	return false
}

func (s *Server) lobbyPlayers() [protocol.QWMaxClients]lobbyPlayer {
	var players [protocol.QWMaxClients]lobbyPlayer
	if s.lobby == nil {
		return players
	}
	for _, c := range s.snapshot() {
		c.mu.Lock()
		players[c.slot] = lobbyPlayer{name: c.name, userID: c.userID}
		c.mu.Unlock()
	}
	return players
}

func (c *client) updateScoreboard(players [protocol.QWMaxClients]lobbyPlayer) []command.Command {
	var commands []command.Command
	for slot, player := range players {
		if c.scoreboard[slot] == player {
			continue
		}
		info := ""
		if player.userID != 0 {
			info = "\\name\\" + player.name
		}
		commands = append(commands, &updateuserinfo.Command{
			PlayerIndex: byte(slot),
			UserID:      player.userID,
			UserInfo:    info,
		})
		c.scoreboard[slot] = player
		// Leave room for chat and sign-on commands in the reliable packet.
		if len(commands) == 8 {
			break
		}
	}
	return commands
}
