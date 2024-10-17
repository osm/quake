package move

import (
	"slices"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/common/crc"
	"github.com/osm/quake/packet/command/deltausercommand"
	"github.com/osm/quake/protocol"
)

type Command struct {
	Checksum byte
	Lossage  byte
	Null     *deltausercommand.Command
	Old      *deltausercommand.Command
	New      *deltausercommand.Command
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.CLCMove)
	buf.PutByte(cmd.Checksum)
	buf.PutByte(cmd.Lossage)

	if cmd.Null != nil {
		buf.PutBytes(cmd.Null.Bytes())
	}

	if cmd.Old != nil {
		buf.PutBytes(cmd.Old.Bytes())
	}

	if cmd.New != nil {
		buf.PutBytes(cmd.New.Bytes())
	}

	return buf.Bytes()
}

func (cmd *Command) GetChecksum(sequence uint32) byte {
	b := slices.Concat(
		[]byte{cmd.Lossage},
		cmd.Null.Bytes(),
		cmd.Old.Bytes(),
		cmd.New.Bytes(),
	)

	return crc.Byte(b, int(sequence))
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.Checksum, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Lossage, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Null, err = deltausercommand.Parse(ctx, buf); err != nil {
		return nil, err
	}

	if cmd.Old, err = deltausercommand.Parse(ctx, buf); err != nil {
		return nil, err
	}

	if cmd.New, err = deltausercommand.Parse(ctx, buf); err != nil {
		return nil, err
	}

	return &cmd, nil
}
