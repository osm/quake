package deltapacketentities

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/packetentity"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Index    byte
	Entities []*packetentity.Command
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCDeltaPacketEntities)
	buf.PutByte(cmd.Index)

	for i := 0; i < len(cmd.Entities); i++ {
		buf.PutBytes(cmd.Entities[i].Bytes())
	}

	buf.PutUint16(0x0000)

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Index, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Entities, err = packetentity.Parse(ctx, buf); err != nil {
		return nil, err
	}

	return &cmd, nil
}
