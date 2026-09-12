package quake

import (
	"fmt"
	"strconv"

	"github.com/osm/quake/common/rand"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/getchallenge"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/command/s2cconnection"
	"github.com/osm/quake/packet/command/stringcmd"
)

func (s *Server) handleClientCommand(client *client, cmd command.Command) []command.Command {
	switch c := cmd.(type) {
	case *stringcmd.Command:
		return s.lobbyCommand(client, c.String)
	case *getchallenge.Command:
		return []command.Command{
			&s2cchallenge.Command{ChallengeID: fmt.Sprintf("c-%d", rand.Uint16())},
		}
	case *connect.Command:
		client.resetSession(localServerPing)
		if s.lobby == nil {
			client.name = c.UserInfo.Get("name")
		}
		client.zExtensions, _ = strconv.Atoi(c.UserInfo.Get("*z_ext"))
		return []command.Command{
			&s2cconnection.Command{},
		}
	}

	return []command.Command{}
}
