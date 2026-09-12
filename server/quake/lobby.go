package quake

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/deltausercommand"
	"github.com/osm/quake/packet/command/download"
	"github.com/osm/quake/packet/command/lightstyle"
	"github.com/osm/quake/packet/command/modellist"
	"github.com/osm/quake/packet/command/packetentities"
	"github.com/osm/quake/packet/command/playerinfo"
	"github.com/osm/quake/packet/command/serverdata"
	"github.com/osm/quake/packet/command/setangle"
	"github.com/osm/quake/packet/command/soundlist"
	"github.com/osm/quake/packet/command/stufftext"
	svctime "github.com/osm/quake/packet/command/time"
	"github.com/osm/quake/packet/command/updateuserinfo"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/zquake"
	"github.com/osm/quake/server"
)

const (
	lobbyIdle = iota
	lobbyServerDataSent
	lobbyPrespawnSent
	lobbySpawnSent
	lobbyActive
)

var lobbyMapName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,48}$`)

// Clients need maps/<Map>.bsp locally because the lobby does not serve downloads.
type Lobby struct {
	Map    string
	Title  string
	Origin [3]float32
	Angles [3]float32
	Stats  func(server.Client) LobbyStats
}

func (s *Server) EnableLobby(lobby Lobby) error {
	if lobby.Map == "" {
		lobby.Map = "start"
	}
	if !lobbyMapName.MatchString(lobby.Map) {
		return fmt.Errorf("quake: invalid lobby map name")
	}

	if lobby.Title == "" {
		lobby.Title = "Quake menu"
	}
	if len(lobby.Title) > 128 || strings.ContainsAny(lobby.Title, "\x00\n\r") || strings.IndexByte(lobby.Title, 0xff) >= 0 {
		return fmt.Errorf("quake: invalid lobby title")
	}

	s.lobby = &lobby

	return nil
}

func (s *Server) lobbyCommand(c *client, text string) []command.Command {
	if s.lobby == nil {
		return nil
	}

	fields := strings.Fields(text)
	if len(fields) == 0 {
		return nil
	}

	l := s.lobby
	switch fields[0] {
	case "new":
		return l.newClient(c)
	case "download":
		return []command.Command{&download.Command{Size16: -1}}
	}

	if c.lobbyStage == lobbyIdle || len(fields) < 2 {
		return nil
	}

	count, err := strconv.Atoi(fields[1])
	if err != nil || count != 1 {
		return nil
	}
	switch fields[0] {
	case "soundlist":
		return []command.Command{&soundlist.Command{ProtocolVersion: protocol.VersionQW}}
	case "modellist":
		return []command.Command{
			&modellist.Command{
				ProtocolVersion: protocol.VersionQW,
				Models:          []string{"maps/" + l.Map + ".bsp"},
			},
		}
	case "prespawn":
		return l.prespawn(c)
	case "spawn":
		return l.spawn(c)
	case "begin":
		if c.lobbyStage != lobbySpawnSent {
			return nil
		}
		c.lobbyStage = lobbyActive
		c.lobbyStarted = time.Now()
	}

	return nil
}

func (l *Lobby) newClient(c *client) []command.Command {
	c.lobbyStage = lobbyServerDataSent
	c.scoreboard = [protocol.QWMaxClients]lobbyPlayer{}
	return []command.Command{
		&serverdata.Command{
			ProtocolVersion: protocol.VersionQW,
			ServerCount:     1,
			GameDirectory:   "qw",
			PlayerNumber:    c.slot,
			LevelName:       l.Title,
			StopSpeed:       100,
			Friction:        4,
			WaterFriction:   1,
		},
		&stufftext.Command{
			String: fmt.Sprintf("fullserverinfo \"\\hostname\\Quake menu\\map\\%s\\*gamedir\\qw\\*z_ext\\%d\"\n", l.Map, c.zExtensions&(zquake.ExtensionPMType|zquake.ExtensionPMTypeNew)),
		},
	}
}

func (l *Lobby) prespawn(c *client) []command.Command {
	if c.lobbyStage != lobbyServerDataSent {
		return nil
	}

	c.lobbyStage = lobbyPrespawnSent
	cmds := make([]command.Command, 0, 65)
	for i := 0; i < 64; i++ {
		cmds = append(cmds, &lightstyle.Command{Index: byte(i), Command: "m"})
	}

	return append(cmds, &stufftext.Command{String: "cmd spawn 1 0\n"})
}

func (l *Lobby) spawn(c *client) []command.Command {
	if c.lobbyStage != lobbyPrespawnSent {
		return nil
	}

	c.lobbyStage = lobbySpawnSent
	info := infostring.New()
	info.Set("name", c.name)

	return []command.Command{
		&updateuserinfo.Command{
			PlayerIndex: c.slot,
			UserID:      c.userID,
			UserInfo:    strings.Trim(string(info.Bytes()), "\""),
		},
		&setangle.Command{Angle: l.Angles},
		&stufftext.Command{String: "skins\n"},
	}
}

func (l *Lobby) frame(c *client) []command.Command {
	return []command.Command{
		&svctime.Command{Time: float32(time.Since(c.lobbyStarted).Seconds() + 1)},
		&playerinfo.Command{
			Index: c.slot,
			Default: &playerinfo.CommandDefault{
				Bits: lobbyPlayerFlags(c),
				DeltaUserCommand: &deltausercommand.Command{
					ProtocolVersion: protocol.VersionQW,
					Bits:            protocol.CMAngle1 | protocol.CMAngle2 | protocol.CMAngle3,
					CMAngle1:        l.Angles[0],
					CMAngle2:        l.Angles[1],
					CMAngle3:        l.Angles[2],
				},
				Coord: l.Origin,
			},
		},
		// Override mouse and keyboard look applied locally by the client.
		&setangle.Command{Angle: l.Angles},
		&packetentities.Command{},
	}
}

func lobbyPlayerFlags(c *client) uint16 {
	flags := uint16(protocol.PFModel | protocol.PFCommand)
	extensions := zquake.ExtensionPMType | zquake.ExtensionPMTypeNew
	if c.zExtensions&extensions == extensions {
		return flags | protocol.PMCLock<<protocol.PFPMCShift
	}

	// Classic clients need dead-player prediction to suppress jumping.
	return flags | protocol.PFDead
}
