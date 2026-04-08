package proxyshell

import (
	"strconv"
	"strings"

	"github.com/osm/quake/packet/clc"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/command/stufftext"
)

func (s *Shell) HandleConnect(
	clientID string,
	cmd *connect.Command,
) bool {
	reset := s.resetIfStale(clientID)
	identity := s.captureShellConnect(clientID, cmd)
	s.logf("%s (%s) connected to shell", identity.name, clientID)

	return reset
}

func (s *Shell) HandleGameData(
	clientID string,
	pkt *clc.GameData,
) []command.Command {
	var out []command.Command

	for _, raw := range pkt.Commands {
		cmd, ok := raw.(*stringcmd.Command)
		if !ok {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(cmd.String), "new") {
			for _, h := range s.newHandlers {
				h(clientID)
			}

			out = append(out, &stufftext.Command{String: "skins"})
			continue
		}

		for _, h := range s.stringHandlers {
			h(clientID, cmd.String)
		}
	}

	return out
}

func (s *Shell) captureShellConnect(
	clientID string,
	cmd *connect.Command,
) identity {
	identity := identity{
		name:          cmd.UserInfo.Get("name"),
		team:          cmd.UserInfo.Get("team"),
		topColor:      byte(atoiLoose(cmd.UserInfo.Get("topcolor"))),
		bottomColor:   byte(atoiLoose(cmd.UserInfo.Get("bottomcolor"))),
		clientVersion: cmd.UserInfo.Get("*client"),
		spectator:     cmd.UserInfo.Get("spectator") == "1",
		userInfo:      cloneInfoString(cmd.UserInfo),
		connect:       cloneConnectCommand(cmd),
	}

	s.mu.Lock()
	s.sessionLocked(clientID).client = identity
	s.mu.Unlock()

	return identity
}

func atoiLoose(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}

	return n
}
