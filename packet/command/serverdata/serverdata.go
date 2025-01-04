package serverdata

import (
	"errors"

	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
	"github.com/osm/quake/protocol/mvd"
)

var (
	ErrUnknownProtocolVersion = errors.New("unknown protocol version")
)

type Command struct {
	IsMVD bool

	ProtocolVersion       uint32
	FTEProtocolExtension  uint32
	FTE2ProtocolExtension uint32
	MVDProtocolExtension  uint32

	// NQ
	MaxClients    byte
	GameType      byte
	SignOnMessage string
	Models        []string
	Sounds        []string

	// QW
	ServerCount       int32
	GameDirectory     string
	LastReceived      float32
	PlayerNumber      byte
	Spectator         bool
	LevelName         string
	Gravity           float32
	StopSpeed         float32
	MaxSpeed          float32
	SpectatorMaxSpeed float32
	Accelerate        float32
	AirAccelerate     float32
	WaterAccelerate   float32
	Friction          float32
	WaterFriction     float32
	EntityGravity     float32
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(protocol.SVCServerData)

	if cmd.FTEProtocolExtension > 0 {
		buf.PutUint32(fte.ProtocolVersion)
		buf.PutUint32(cmd.FTEProtocolExtension)
	}

	if cmd.FTE2ProtocolExtension > 0 {
		buf.PutUint32(fte.ProtocolVersion2)
		buf.PutUint32(cmd.FTE2ProtocolExtension)
	}

	if cmd.MVDProtocolExtension > 0 {
		buf.PutUint32(mvd.ProtocolVersion)
		buf.PutUint32(cmd.MVDProtocolExtension)
	}

	buf.PutUint32(uint32(cmd.ProtocolVersion))

	if cmd.ProtocolVersion == protocol.VersionNQ {
		buf.PutByte(cmd.MaxClients)
		buf.PutByte(cmd.GameType)
		buf.PutString(cmd.SignOnMessage)

		for i := 0; i < len(cmd.Models); i++ {
			buf.PutString(cmd.Models[i])
		}
		buf.PutByte(0x00)

		for i := 0; i < len(cmd.Sounds); i++ {
			buf.PutString(cmd.Sounds[i])
		}
		buf.PutByte(0x00)
	} else {
		buf.PutUint32(uint32(cmd.ServerCount))
		buf.PutString(cmd.GameDirectory)

		if cmd.IsMVD {
			buf.PutFloat32(cmd.LastReceived)
		} else {
			buf.PutByte(cmd.PlayerNumber)
		}

		buf.PutString(cmd.LevelName)

		if cmd.ProtocolVersion >= 25 {
			buf.PutFloat32(cmd.Gravity)
			buf.PutFloat32(cmd.StopSpeed)
			buf.PutFloat32(cmd.MaxSpeed)
			buf.PutFloat32(cmd.SpectatorMaxSpeed)
			buf.PutFloat32(cmd.Accelerate)
			buf.PutFloat32(cmd.AirAccelerate)
			buf.PutFloat32(cmd.WaterAccelerate)
			buf.PutFloat32(cmd.Friction)
			buf.PutFloat32(cmd.WaterFriction)
			buf.PutFloat32(cmd.EntityGravity)
		}
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.IsMVD = ctx.GetIsMVD()

	for {
		pv, err := buf.GetUint32()
		if err != nil {
			return nil, err
		}

		if pv == fte.ProtocolVersion {
			if cmd.FTEProtocolExtension, err = buf.GetUint32(); err != nil {
				return nil, err
			}
			ctx.SetFTEProtocolExtension(cmd.FTEProtocolExtension)
			if cmd.FTEProtocolExtension&fte.ExtensionFloatCoords != 0 {
				ctx.SetAngleSize(2)
				ctx.SetCoordSize(4)
			}
			continue
		}

		if pv == fte.ProtocolVersion2 {
			if cmd.FTE2ProtocolExtension, err = buf.GetUint32(); err != nil {
				return nil, err
			}
			ctx.SetFTE2ProtocolExtension(cmd.FTE2ProtocolExtension)
			continue
		}

		if pv == mvd.ProtocolVersion {
			if cmd.MVDProtocolExtension, err = buf.GetUint32(); err != nil {
				return nil, err
			}
			ctx.SetMVDProtocolExtension(cmd.MVDProtocolExtension)
			continue
		}

		if pv == protocol.VersionNQ || pv == protocol.VersionQW || pv == protocol.VersionQW210 {
			cmd.ProtocolVersion = pv
			ctx.SetProtocolVersion(cmd.ProtocolVersion)
			break
		}

		return nil, ErrUnknownProtocolVersion
	}

	if cmd.ProtocolVersion == protocol.VersionNQ {
		if err = parseCommandNQ(ctx, buf, &cmd); err != nil {
			return nil, err
		}
	} else {
		if err = parseCommandQW(ctx, buf, &cmd); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}

func parseCommandNQ(ctx *context.Context, buf *buffer.Buffer, cmd *Command) error {
	var err error

	if cmd.MaxClients, err = buf.ReadByte(); err != nil {
		return err
	}

	if cmd.GameType, err = buf.ReadByte(); err != nil {
		return err
	}

	if cmd.SignOnMessage, err = buf.GetString(); err != nil {
		return err
	}

	for {
		var model string
		if model, err = buf.GetString(); err != nil {
			return err
		}

		if model == "" {
			break
		}

		cmd.Models = append(cmd.Models, model)
	}

	for {
		var sound string
		if sound, err = buf.GetString(); err != nil {
			return err
		}

		if sound == "" {
			break
		}

		cmd.Sounds = append(cmd.Sounds, sound)
	}

	return nil
}

func parseCommandQW(ctx *context.Context, buf *buffer.Buffer, cmd *Command) error {
	var err error

	if cmd.ServerCount, err = buf.GetInt32(); err != nil {
		return err
	}

	if cmd.GameDirectory, err = buf.GetString(); err != nil {
		return err
	}

	if cmd.IsMVD {
		if cmd.LastReceived, err = buf.GetFloat32(); err != nil {
			return err
		}
	} else {
		if cmd.PlayerNumber, err = buf.ReadByte(); err != nil {
			return err
		}
		if cmd.PlayerNumber&128 != 0 {
			cmd.Spectator = true
		}
	}

	if cmd.LevelName, err = buf.GetString(); err != nil {
		return err
	}

	if cmd.ProtocolVersion >= 25 {
		if cmd.Gravity, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.StopSpeed, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.MaxSpeed, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.SpectatorMaxSpeed, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.Accelerate, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.AirAccelerate, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.WaterAccelerate, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.Friction, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.WaterFriction, err = buf.GetFloat32(); err != nil {
			return err
		}
		if cmd.EntityGravity, err = buf.GetFloat32(); err != nil {
			return err
		}
	}

	return nil
}
