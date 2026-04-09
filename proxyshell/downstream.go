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

func (s *Shell) ProcessShellConnect(
	clientID string,
	cmd *connect.Command,
) (bool, bool) {
	reset := s.resetIfStale(clientID)
	identity := s.captureShellConnect(clientID, cmd)
	s.logf("%s (%s) connected to shell", identity.name, clientID)

	for _, h := range s.shellPacketHandlers {
		h(clientID, &clc.Connectionless{Command: cmd})
	}

	s.mu.Lock()
	hasTarget := s.sessionLocked(clientID).target != ""
	s.mu.Unlock()
	s.ensureRemoteConnected(clientID)

	return reset, hasTarget
}

func (s *Shell) ProcessShellGameData(
	clientID string,
	pkt *clc.GameData,
) []command.Command {
	for _, h := range s.shellPacketHandlers {
		h(clientID, pkt)
	}

	var out []command.Command

	for _, raw := range pkt.Commands {
		cmd, ok := raw.(*stringcmd.Command)
		if !ok {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(cmd.String), "new") {
			out = append(out, &stufftext.Command{String: "skins"})
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
