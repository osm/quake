package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
)

const standardBindings = `bind uparrow "say proxy:menu up"
bind downarrow "say proxy:menu down"
bind leftarrow "say proxy:menu left"
bind rightarrow "say proxy:menu right"
bind enter "say proxy:menu select"
bind del "say proxy:menu delete"
bind home "say proxy:menu home"
bind end "say proxy:menu end"
bind pgup "say proxy:menu pgup"
bind pgdn "say proxy:menu pgdn"
bind backspace "say proxy:menu back"
bind ins "say proxy:menu help"
bind pause "say proxy:menu"
`

func (c *Connection) bindStandard() error {
	text := strings.ReplaceAll(standardBindings, "proxy:menu", c.options.Namespace)
	cmd := &stufftext.Command{String: text}
	if 8+len(cmd.Bytes()) > c.options.MaxPacketBytes {
		return fmt.Errorf("ui: standard bindings exceed packet limit")
	}
	c.bindings = cmd
	c.bindSequences = nil
	return nil
}

func (c *Connection) appendBindings(p *svc.GameData, now time.Time) {
	if c.bindings == nil {
		return
	}
	if len(c.bindSequences) > 0 && now.Sub(c.lastBind) < c.options.RefreshInterval {
		return
	}
	if packetSize(p)+len(c.bindings.Bytes()) > c.options.MaxPacketBytes {
		return
	}

	p.Commands = append(p.Commands, c.bindings)
	c.lastBind = now
	if len(c.bindSequences) >= 16 {
		c.bindSequences = c.bindSequences[1:]
	}
	c.bindSequences = append(c.bindSequences, p.Seq&protocol.QWSequenceMask)
}
