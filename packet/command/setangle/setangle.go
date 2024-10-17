package setangle

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/mvd"
)

type Command struct {
	AngleSize            uint8
	MVDProtocolExtension uint32
	IsMVD                bool

	MVDAngleIndex byte
	Angle         [3]float32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	writeAngle := buf.PutAngle8
	if cmd.AngleSize == 2 {
		writeAngle = buf.PutAngle16
	}

	buf.PutByte(protocol.SVCSetAngle)

	if cmd.IsMVD || cmd.MVDProtocolExtension&mvd.ExtensionHighLagTeleport != 0 {
		buf.PutByte(cmd.MVDAngleIndex)
	}

	for i := 0; i < 3; i++ {
		writeAngle(cmd.Angle[i])
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.MVDProtocolExtension = ctx.GetMVDProtocolExtension()
	cmd.IsMVD = ctx.GetIsMVD()

	cmd.AngleSize = ctx.GetAngleSize()
	readAngle := buf.GetAngle8
	if cmd.AngleSize == 2 {
		readAngle = buf.GetAngle16
	}

	if cmd.IsMVD || cmd.MVDProtocolExtension&mvd.ExtensionHighLagTeleport != 0 {
		if cmd.MVDAngleIndex, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	for i := 0; i < 3; i++ {
		if cmd.Angle[i], err = readAngle(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
