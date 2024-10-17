package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type Set struct {
	SeqOut uint32
	SeqIn  uint32
}

func (cmd *Set) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint32(cmd.SeqOut)
	buf.PutUint32(cmd.SeqIn)

	return buf.Bytes()
}

func parseSet(ctx *context.Context, buf *buffer.Buffer) (*Set, error) {
	var err error
	var cmd Set

	if cmd.SeqOut, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	if cmd.SeqIn, err = buf.GetUint32(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
