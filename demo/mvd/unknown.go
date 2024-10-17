package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type Unknown struct {
	Data []byte
}

func (cmd *Unknown) Bytes() []byte {
	return cmd.Data
}

func parseUnknown(ctx *context.Context, buf *buffer.Buffer, size uint32) (*Unknown, error) {
	var err error
	var cmd Unknown

	if cmd.Data, err = buf.GetBytes(int(size)); err != nil {
		return nil, err
	}

	return &cmd, nil
}
