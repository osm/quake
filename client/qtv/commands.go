package qtv

import (
	"fmt"

	"github.com/osm/quake/common/args"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/modellist"
	"github.com/osm/quake/packet/command/qtvstringcmd"
	"github.com/osm/quake/packet/command/serverdata"
	"github.com/osm/quake/packet/command/soundlist"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/svc"
)

func (c *Client) handleGameData(gameData *svc.GameData) []command.Command {
	var cmds []command.Command

	for _, cmd := range gameData.Commands {
		switch cmd := cmd.(type) {
		case *serverdata.Command:
			cmds = append(cmds, c.handleServerData(cmd)...)
		case *soundlist.Command:
			cmds = append(cmds, c.handleSoundList(cmd)...)
		case *modellist.Command:
			cmds = append(cmds, c.handleModelList(cmd)...)
		case *stufftext.Command:
			cmds = append(cmds, c.handleStufftext(cmd)...)
		}
	}

	return cmds
}

func (c *Client) handleServerData(cmd *serverdata.Command) []command.Command {
	c.serverCount = cmd.ServerCount

	return []command.Command{
		&qtvstringcmd.Command{String: fmt.Sprintf("qtvsoundlist %d 0", c.serverCount)},
	}
}

func (c *Client) handleSoundList(cmd *soundlist.Command) []command.Command {
	if cmd.Index > 0 {
		return nil
	}

	return []command.Command{
		&qtvstringcmd.Command{String: fmt.Sprintf("qtvmodellist %d 0", c.serverCount)},
	}
}

func (c *Client) handleModelList(cmd *modellist.Command) []command.Command {
	if cmd.Index > 0 {
		return nil
	}

	return []command.Command{
		&qtvstringcmd.Command{String: fmt.Sprintf("qtvspawn %d", c.serverCount)},
	}
}

func (c *Client) handleStufftext(cmd *stufftext.Command) []command.Command {
	for _, a := range args.Parse(cmd.String) {
		switch a.Cmd {
		case "skins":
			c.isConnected = true
		}
	}

	return nil
}
