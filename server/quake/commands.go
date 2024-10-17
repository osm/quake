package quake

import (
	"fmt"

	"github.com/osm/quake/common/rand"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/getchallenge"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/command/s2cconnection"
)

func (s *Server) handleClientCommand(client *client, cmd command.Command) []command.Command {
	switch c := cmd.(type) {
	case *getchallenge.Command:
		return []command.Command{
			&s2cchallenge.Command{ChallengeID: fmt.Sprintf("c-%d", rand.Uint16())},
		}
	case *connect.Command:
		client.name = c.UserInfo.Get("name")
		return []command.Command{
			&s2cconnection.Command{},
		}
	}

	return []command.Command{}
}
