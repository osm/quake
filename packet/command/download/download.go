package download

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
)

type Command struct {
	FTEProtocolExtension uint32

	Size16  int16
	Size32  int32
	Percent byte
	Number  int32
	Name    string
	Data    []byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCDownload)

	if cmd.FTEProtocolExtension&fte.ExtensionChunkedDownloads != 0 {
		buf.PutInt32(cmd.Number)

		if cmd.Number == -1 {
			buf.PutInt32(cmd.Size32)
			buf.PutString(cmd.Name)
		} else {
			buf.PutBytes(cmd.Data)
		}
	} else {
		buf.PutInt16(cmd.Size16)
		buf.PutByte(cmd.Percent)

		if cmd.Size16 != -1 {
			buf.PutBytes(cmd.Data)
		}
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.FTEProtocolExtension = ctx.GetFTEProtocolExtension()

	if cmd.FTEProtocolExtension&fte.ExtensionChunkedDownloads != 0 {
		if cmd.Number, err = buf.GetInt32(); err != nil {
			return nil, err
		}

		if cmd.Number < 0 {
			if cmd.Size32, err = buf.GetInt32(); err != nil {
				return nil, err
			}

			if cmd.Name, err = buf.GetString(); err != nil {
				return nil, err
			}

			return &cmd, nil
		}

		if cmd.Data, err = buf.GetBytes(protocol.DownloadBlockSize); err != nil {
			return nil, err
		}
	} else {
		if cmd.Size16, err = buf.GetInt16(); err != nil {
			return nil, err
		}

		if cmd.Percent, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		if cmd.Size16 > 0 {
			if cmd.Data, err = buf.GetBytes(int(cmd.Size16)); err != nil {
				return nil, err
			}
		}
	}

	return &cmd, nil
}
