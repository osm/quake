package mvd

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
)

type DamageDone struct {
	size uint32

	Data        []byte
	DeathType   uint16
	AttackerEnt uint16
	TargetEnt   uint16
	Damage      uint16
}

func (cmd *DamageDone) Bytes() []byte {
	if cmd.size != 8 {
		return cmd.Data
	}

	buf := buffer.New()
	buf.PutUint16(cmd.DeathType)
	buf.PutUint16(cmd.AttackerEnt)
	buf.PutUint16(cmd.TargetEnt)
	buf.PutUint16(cmd.Damage)

	return buf.Bytes()
}

func parseDamageDone(ctx *context.Context, buf *buffer.Buffer, size uint32) (*DamageDone, error) {
	var err error
	var cmd DamageDone

	cmd.size = size
	if cmd.size != 8 {
		if cmd.Data, err = buf.GetBytes(int(cmd.size)); err != nil {
			return nil, err
		}

		goto end
	}

	if cmd.DeathType, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.AttackerEnt, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.TargetEnt, err = buf.GetUint16(); err != nil {
		return nil, err
	}

	if cmd.Damage, err = buf.GetUint16(); err != nil {
		return nil, err
	}

end:
	return &cmd, nil
}
