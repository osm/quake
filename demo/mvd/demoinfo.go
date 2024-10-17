package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type DemoInfo struct {
	BlockNumber uint16
	Data        []byte
}

func (cmd *DemoInfo) Bytes() []byte {
	buf := buffer.New()

	buf.PutUint16(cmd.BlockNumber)
	buf.PutBytes(cmd.Data)

	return buf.Bytes()
}

func parseDemoInfo(ctx *context.Context, buf *buffer.Buffer, size uint32) (*DemoInfo, error) {
	var err error
	var cmd DemoInfo

	if cmd.BlockNumber, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Data, err = buf.GetBytes(int(size) - 2); err != nil {
		return nil, err
	}

	return &cmd, nil
}
