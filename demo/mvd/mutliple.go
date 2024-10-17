package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol/mvd"
)

type Multiple struct {
	LastTo         uint32
	IsHiddenPacket bool
	Size           uint32
	HiddenCommands []*HiddenCommand
}

func (cmd *Multiple) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint32(cmd.LastTo)

	if cmd.IsHiddenPacket {
		buf.PutUint32(cmd.Size)

		for _, c := range cmd.HiddenCommands {
			buf.PutBytes(c.Bytes())
		}
	}

	return buf.Bytes()
}

func parseMultiple(ctx *context.Context, buf *buffer.Buffer) (*Multiple, error) {
	var err error
	var cmd Multiple

	if cmd.LastTo, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if cmd.LastTo == 0 && ctx.GetMVDProtocolExtension()&mvd.ExtensionHiddenMessages != 0 {
		cmd.IsHiddenPacket = true

		if cmd.Size, err = buf.GetUint32(); err != nil {
			return nil, err
		}

		bytes, err := buf.GetBytes(int(cmd.Size))
		if err != nil {
			return nil, err
		}

		if cmd.HiddenCommands, err = parseHiddenCommands(ctx, bytes); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
