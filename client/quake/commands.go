package quake

import (
	"fmt"
	"strings"
	"time"

	"github.com/osm/quake/common/args"
	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/common/sequencer"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/connect"
	"github.com/osm/quake/packet/command/ip"
	"github.com/osm/quake/packet/command/modellist"
	"github.com/osm/quake/packet/command/s2cchallenge"
	"github.com/osm/quake/packet/command/s2cconnection"
	"github.com/osm/quake/packet/command/serverdata"
	"github.com/osm/quake/packet/command/soundlist"
	"github.com/osm/quake/packet/command/stringcmd"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
	"github.com/osm/quake/protocol/mvd"
)

func (c *Client) handleServerCommand(cmd command.Command) []command.Command {
	switch cmd := cmd.(type) {
	case *s2cchallenge.Command:
		return c.handleChallenge(cmd)
	case *s2cconnection.Command:
		c.seq.SetState(sequencer.Connecting)
		return c.handleNew()
	case *stufftext.Command:
		return c.handleStufftext(cmd)
	case *soundlist.Command:
		return c.handleSoundList(cmd)
	case *modellist.Command:
		return c.handleModelList(cmd)
	case *serverdata.Command:
		c.serverCount = cmd.ServerCount
	}

	return nil
}

func (c *Client) handleChallenge(cmd *s2cchallenge.Command) []command.Command {
	userInfo := infostring.New(
		infostring.WithKeyValue("name", c.name),
		infostring.WithKeyValue("team", c.team),
		infostring.WithKeyValue("topcolor", fmt.Sprintf("%d", c.topColor)),
		infostring.WithKeyValue("bottomcolor", fmt.Sprintf("%d", c.bottomColor)),
	)

	if c.addrPort != "" {
		userInfo.Info = append(userInfo.Info, infostring.Info{
			Key:   "prx",
			Value: c.addrPort,
		})
	}

	if c.clientVersion != "" {
		userInfo.Info = append(userInfo.Info, infostring.Info{
			Key:   "*client",
			Value: c.clientVersion,
		})
	}

	if c.isSpectator {
		userInfo.Info = append(userInfo.Info, infostring.Info{
			Key:   "spectator",
			Value: "1",
		})
	}

	if c.zQuakeEnabled {
		userInfo.Info = append(userInfo.Info, infostring.Info{
			Key:   "*z_ext",
			Value: fmt.Sprintf("%d", c.zQuakeExtensions),
		})
	}

	var extensions []*protocol.Extension
	for _, ext := range cmd.Extensions {
		if c.fteEnabled && ext.Version == fte.ProtocolVersion {
			extensions = append(extensions, ext)
		} else if c.fte2Enabled && ext.Version == fte.ProtocolVersion2 {
			extensions = append(extensions, ext)
		} else if c.mvdEnabled && ext.Version == mvd.ProtocolVersion {
			extensions = append(extensions, ext)
		}
	}

	return []command.Command{
		&connect.Command{
			Command:     "connect",
			Version:     fmt.Sprintf("%v", c.ctx.GetProtocolVersion()),
			QPort:       c.qPort,
			ChallengeID: cmd.ChallengeID,
			UserInfo:    userInfo,
			Extensions:  extensions,
		},
	}
}

func (c *Client) handleNew() []command.Command {
	return []command.Command{&stringcmd.Command{String: "new"}}
}

func (c *Client) handleStufftext(cmd *stufftext.Command) []command.Command {
	var cmds []command.Command

	for _, a := range args.Parse(cmd.String) {
		switch a.Cmd {
		case "reconnect":
			c.seq.SetState(sequencer.Connecting)
			cmds = append(cmds, c.handleNew()...)
		case "cmd":
			cmds = append(cmds, c.handleStufftextCmds(a.Args)...)
		case "packet":
			cmds = append(cmds, &ip.Command{String: a.Args[1][1 : len(a.Args[1])-1]})
		case "fullserverinfo":
			cmds = append(cmds, c.handleFullServerInfo(a.Args[0])...)
		case "skins":
			c.readDeadline = time.Duration(c.ping)
			c.seq.SetState(sequencer.Connected)
			cmds = append(cmds, &stringcmd.Command{
				String: fmt.Sprintf("begin %d", c.serverCount),
			})
		}
	}

	return cmds
}

func (c *Client) handleStufftextCmds(args []string) []command.Command {
	switch args[0] {
	case "ack", "prespawn", "spawn":
		return []command.Command{&stringcmd.Command{String: strings.Join(args, " ")}}
	case "pext":
		return c.handleProtocolExtensions()
	case "new":
		return c.handleNew()
	}

	return nil
}

func (c *Client) handleProtocolExtensions() []command.Command {
	var fteVersion uint32
	var fteExtensions uint32
	var fte2Version uint32
	var fte2Extensions uint32
	var mvdVersion uint32
	var mvdExtensions uint32

	if c.fteEnabled {
		fteVersion = fte.ProtocolVersion
		fteExtensions = c.fteExtensions
	}
	if c.fte2Enabled {
		fte2Version = fte.ProtocolVersion2
		fte2Extensions = c.fte2Extensions
	}
	if c.mvdEnabled {
		mvdVersion = mvd.ProtocolVersion
		mvdExtensions = c.mvdExtensions
	}

	return []command.Command{
		&stringcmd.Command{String: fmt.Sprintf("pext 0x%x 0x%x 0x%x 0x%x 0x%x 0x%x",
			fteVersion,
			fteExtensions,
			fte2Version,
			fte2Extensions,
			mvdVersion,
			mvdExtensions,
		)},
	}
}

func (c *Client) handleFullServerInfo(args string) []command.Command {
	inf := infostring.Parse(args)
	c.mapName = inf.Get("map")

	return []command.Command{
		&stringcmd.Command{String: fmt.Sprintf("soundlist %d 0", c.serverCount)},
	}
}

func (c *Client) handleSoundList(cmd *soundlist.Command) []command.Command {
	nextCmd := "soundlist"
	if cmd.Index == 0 {
		nextCmd = "modellist"
	}

	return []command.Command{
		&stringcmd.Command{
			String: fmt.Sprintf("%s %d %d", nextCmd, c.serverCount, cmd.Index),
		},
	}
}

func (c *Client) handleModelList(cmd *modellist.Command) []command.Command {
	if cmd.Index == 0 {
		return c.handlePrespawn()
	}

	return []command.Command{
		&stringcmd.Command{
			String: fmt.Sprintf("modellist %d %d", c.serverCount, cmd.Index),
		},
	}
}

func (c *Client) handlePrespawn() []command.Command {
	hash, ok := protocol.MapChecksum[c.mapName]
	if !ok {
		c.logger.Printf("missing map checksum for %s", c.mapName)
	}

	return []command.Command{
		&stringcmd.Command{String: fmt.Sprintf("setinfo pmodel %d", protocol.PlayerModel)},
		&stringcmd.Command{String: fmt.Sprintf("setinfo emodel %d", protocol.EyeModel)},
		&stringcmd.Command{String: fmt.Sprintf("prespawn %v 0 %d", c.serverCount, hash)},
	}
}
