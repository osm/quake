package s2cchallenge

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
	"github.com/osm/quake/protocol/mvd"
)

type Command struct {
	ChallengeID string
	Extensions  []*protocol.Extension
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(byte(protocol.S2CChallenge))
	buf.PutString(cmd.ChallengeID)

	for _, ext := range cmd.Extensions {
		buf.PutBytes(ext.Bytes())
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	if cmd.ChallengeID, err = buf.GetString(); err != nil {
		return nil, err
	}

	for buf.Off() < buf.Len() {
		version, err := buf.GetUint32()
		if err != nil {
			return nil, err
		}

		extensions, err := buf.GetUint32()
		if err != nil {
			return nil, err
		}

		if ctx.GetIsFTEEnabled() && version == fte.ProtocolVersion {
			ctx.SetFTEProtocolExtension(extensions)

			if extensions&fte.ExtensionFloatCoords != 0 {
				ctx.SetAngleSize(1)
				ctx.SetCoordSize(2)
			}
		}

		if ctx.GetIsFTE2Enabled() && version == fte.ProtocolVersion2 {
			ctx.SetFTE2ProtocolExtension(extensions)
		}

		if ctx.GetIsMVDEnabled() && version == mvd.ProtocolVersion {
			ctx.SetMVDProtocolExtension(extensions)
		}

		cmd.Extensions = append(cmd.Extensions, &protocol.Extension{
			Version:    version,
			Extensions: extensions,
		})
	}

	return &cmd, nil
}
