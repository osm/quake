package mvd

import (
	"errors"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/mvd"
)

var ErrUnknownType = errors.New("unknown type")

type Demo struct {
	Data []Data
}

type Data struct {
	Timestamp byte
	Command   byte
	Cmd       *Cmd
	Read      *Read
	Set       *Set
	Multiple  *Multiple
}

func (d *Demo) Bytes() []byte {
	buf := buffer.New()

	for i := 0; i < len(d.Data); i++ {
		buf.PutBytes(d.Data[i].Bytes())
	}

	return buf.Bytes()
}

func (d *Data) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(d.Timestamp)
	buf.PutByte(d.Command)

	switch d.Command & 0x7 {
	case mvd.DemoMultiple:
		buf.PutBytes(d.Multiple.Bytes())
		fallthrough
	case mvd.DemoStats:
		fallthrough
	case mvd.DemoSingle:
		fallthrough
	case mvd.DemoAll:
		fallthrough
	case protocol.DemoRead:
		buf.PutBytes(d.Read.Bytes())
	case protocol.DemoSet:
		buf.PutBytes(d.Set.Bytes())
	case protocol.DemoCmd:
		buf.PutBytes(d.Cmd.Bytes())
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, data []byte) (*Demo, error) {
	var err error
	var cmd Demo

	buf := buffer.New(buffer.WithData(data))
	ctx.SetIsMVD(true)

	for buf.Off() < buf.Len() {
		var data Data

	process:
		if data.Timestamp, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		if data.Command, err = buf.ReadByte(); err != nil {
			return nil, err
		}

		switch data.Command & 0x7 {
		case mvd.DemoMultiple:
			if data.Multiple, err = parseMultiple(ctx, buf); err != nil {
				return nil, err
			}

			if data.Multiple.IsHiddenPacket {
				cmd.Data = append(cmd.Data, data)

				if buf.Off() == buf.Len() {
					goto end
				}

				goto process
			}

			fallthrough
		case mvd.DemoStats:
			fallthrough
		case mvd.DemoSingle:
			fallthrough
		case mvd.DemoAll:
			fallthrough
		case protocol.DemoRead:
			if data.Read, err = parseRead(ctx, buf); err != nil {
				return nil, err
			}
		case protocol.DemoSet:
			if data.Set, err = parseSet(ctx, buf); err != nil {
				return nil, err
			}
		case protocol.DemoCmd:
			if data.Cmd, err = parseCmd(ctx, buf); err != nil {
				return nil, err
			}
		default:
			return nil, ErrUnknownType
		}

		cmd.Data = append(cmd.Data, data)
	}

end:
	return &cmd, nil
}
