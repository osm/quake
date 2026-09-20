package mvd

import (
	"errors"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/packet/command/disconnect"
	"github.com/osm/quake/packet/svc"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/mvd"
)

var ErrUnknownType = errors.New("unknown type")

type Demo struct {
	Data         []Data
	TrailingData []byte
}

type Data struct {
	Target    uint32
	Timestamp byte
	Command   byte
	Cmd       *Cmd
	Read      *Read
	Set       *Set
	Multiple  *Multiple
	original  []byte
}

func (d *Demo) Bytes() []byte {
	buf := buffer.New()

	for i := 0; i < len(d.Data); i++ {
		buf.PutBytes(d.Data[i].Bytes())
	}
	buf.PutBytes(d.TrailingData)

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

// OriginalBytes returns the exact record bytes consumed by Parse. It returns
// nil for Data values that were not produced by Parse. Callers must not modify
// the returned storage.
func (d *Data) OriginalBytes() []byte {
	return d.original
}

func Parse(ctx *context.Context, data []byte) (*Demo, error) {
	var cmd Demo
	trailing, err := ParseRecords(ctx, data, func(data *Data) error {
		cmd.Data = append(cmd.Data, *data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	cmd.TrailingData = append([]byte(nil), trailing...)
	return &cmd, nil
}

// ParseRecords parses an MVD and visits each record in wire order. The Data
// value and returned trailing bytes refer to the input storage and must not be
// modified. Callers that need to retain parsed records should copy the Data
// value, as Parse does.
func ParseRecords(
	ctx *context.Context,
	input []byte,
	visit func(*Data) error,
) ([]byte, error) {
	if visit == nil {
		return nil, errors.New("nil MVD record visitor")
	}
	var err error

	buf := buffer.New(buffer.WithData(input))
	ctx.SetIsMVD(true)

	for buf.Off() < buf.Len() {
		var data Data

	process:
		start := buf.Off()
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
			data.Target = data.Multiple.LastTo

			if data.Multiple.IsHiddenPacket {
				data.original = buf.Bytes()[start:buf.Off()]
				if err := visit(&data); err != nil {
					return nil, err
				}

				if buf.Off() == buf.Len() {
					return nil, nil
				}

				goto process
			}

			fallthrough
		case mvd.DemoStats:
			fallthrough
		case mvd.DemoSingle:
			// Target determines which client the data is intended
			// to reach and can be used in conjunction with the
			// updateuserinfo to determine who the client is.
			data.Target = uint32(data.Command >> 3)
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

		data.original = buf.Bytes()[start:buf.Off()]
		if err := visit(&data); err != nil {
			return nil, err
		}
		if data.endsDemo() {
			return buf.Bytes()[buf.Off():], nil
		}
	}

	return nil, nil
}

func (d Data) endsDemo() bool {
	if d.Read == nil {
		return false
	}

	gd, ok := d.Read.Packet.(*svc.GameData)
	if !ok {
		return false
	}

	for _, cmd := range gd.Commands {
		if _, ok := cmd.(*disconnect.Command); ok {
			return true
		}
	}

	return false
}
