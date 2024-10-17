package quake

import (
	"math/rand"

	"github.com/osm/quake/packet/command/deltausercommand"
	"github.com/osm/quake/packet/command/move"
	"github.com/osm/quake/protocol"
)

func (c *Client) getMove(seq uint32) *move.Command {
	mov := move.Command{
		Null: &deltausercommand.Command{
			ProtocolVersion: c.ctx.GetProtocolVersion(),
			Bits:            protocol.CMForward | protocol.CMSide | protocol.CMUp | protocol.CMButtons,
			CMForward16:     uint16(rand.Uint32() & 0xffff),
			CMSide16:        uint16(rand.Uint32() & 0xffff),
			CMUp16:          uint16(rand.Uint32() & 0xffff),
			CMButtons:       protocol.ButtonAttack,
			CMMsec:          byte(rand.Intn(255-50+1) + 50),
		},
		Old: &deltausercommand.Command{ProtocolVersion: c.ctx.GetProtocolVersion()},
		New: &deltausercommand.Command{ProtocolVersion: c.ctx.GetProtocolVersion()},
	}

	mov.Checksum = mov.GetChecksum(seq - 1)
	return &mov
}
