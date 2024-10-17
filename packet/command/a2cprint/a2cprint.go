package a2cprint

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/ftedownload"
	"github.com/osm/quake/protocol"
)

type Command struct {
	String string

	IsChunkedDownload bool
	Download          *ftedownload.Command
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.A2CPrint)

	if cmd.IsChunkedDownload && cmd.Download != nil {
		buf.PutBytes(cmd.Download.Bytes())
	} else {
		buf.PutString(cmd.String)
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	bytes, _ := buf.PeekBytes(6)
	cmd.IsChunkedDownload = bytes != nil && string(bytes) == "\\chunk"

	if cmd.IsChunkedDownload {
		cmd.IsChunkedDownload = true

		if cmd.Download, err = ftedownload.Parse(ctx, buf); err != nil {
			return nil, err
		}

	}

	if cmd.String, err = buf.GetString(); err != nil {
		return nil, err
	}

	return &cmd, nil
}
