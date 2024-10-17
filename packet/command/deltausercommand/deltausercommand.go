package deltausercommand

import (
	"github.com/osm/quake/common/buffer"
	"github.com/osm/quake/common/context"
	"github.com/osm/quake/protocol"
)

type Command struct {
	ProtocolVersion uint32

	Bits byte

	CMAngle1    float32
	CMAngle2    float32
	CMAngle3    float32
	CMForward8  byte
	CMForward16 uint16
	CMSide8     byte
	CMSide16    uint16
	CMUp8       byte
	CMUp16      uint16
	CMButtons   byte
	CMImpulse   byte
	CMMsec      byte
}

func (cmd *Command) Bytes() []byte {
	buf := buffer.New()

	buf.PutByte(cmd.Bits)

	if cmd.Bits&protocol.CMAngle1 != 0 {
		buf.PutAngle16(cmd.CMAngle1)
	}

	if cmd.Bits&protocol.CMAngle2 != 0 {
		buf.PutAngle16(cmd.CMAngle2)
	}

	if cmd.Bits&protocol.CMAngle3 != 0 {
		buf.PutAngle16(cmd.CMAngle3)
	}

	if cmd.ProtocolVersion <= 26 {
		if cmd.Bits&protocol.CMForward != 0 {
			buf.PutByte(cmd.CMForward8)
		}

		if cmd.Bits&protocol.CMSide != 0 {
			buf.PutByte(cmd.CMSide8)
		}

		if cmd.Bits&protocol.CMUp != 0 {
			buf.PutByte(cmd.CMUp8)
		}
	} else {
		if cmd.Bits&protocol.CMForward != 0 {
			buf.PutUint16(cmd.CMForward16)
		}

		if cmd.Bits&protocol.CMSide != 0 {
			buf.PutUint16(cmd.CMSide16)
		}

		if cmd.Bits&protocol.CMUp != 0 {
			buf.PutUint16(cmd.CMUp16)
		}
	}

	if cmd.Bits&protocol.CMButtons != 0 {
		buf.PutByte(cmd.CMButtons)
	}

	if cmd.Bits&protocol.CMImpulse != 0 {
		buf.PutByte(cmd.CMImpulse)
	}

	if cmd.ProtocolVersion <= 26 && cmd.Bits&protocol.CMMsec != 0 {
		buf.PutByte(cmd.CMMsec)
	} else if cmd.ProtocolVersion >= 27 {
		buf.PutByte(cmd.CMMsec)
	}

	return buf.Bytes()
}

func Parse(ctx *context.Context, buf *buffer.Buffer) (*Command, error) {
	var err error
	var cmd Command

	cmd.ProtocolVersion = ctx.GetProtocolVersion()

	if cmd.Bits, err = buf.ReadByte(); err != nil {
		return nil, err
	}

	if cmd.Bits&protocol.CMAngle1 != 0 {
		if cmd.CMAngle1, err = buf.GetAngle16(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.CMAngle2 != 0 {
		if cmd.CMAngle2, err = buf.GetAngle16(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.CMAngle3 != 0 {
		if cmd.CMAngle3, err = buf.GetAngle16(); err != nil {
			return nil, err
		}
	}

	if cmd.ProtocolVersion <= 26 {
		if cmd.Bits&protocol.CMForward != 0 {
			if cmd.CMForward8, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}

		if cmd.Bits&protocol.CMSide != 0 {
			if cmd.CMSide8, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}

		if cmd.Bits&protocol.CMUp != 0 {
			if cmd.CMUp8, err = buf.ReadByte(); err != nil {
				return nil, err
			}
		}
	} else {
		if cmd.Bits&protocol.CMForward != 0 {
			if cmd.CMForward16, err = buf.GetUint16(); err != nil {
				return nil, err
			}
		}

		if cmd.Bits&protocol.CMSide != 0 {
			if cmd.CMSide16, err = buf.GetUint16(); err != nil {
				return nil, err
			}
		}

		if cmd.Bits&protocol.CMUp != 0 {
			if cmd.CMUp16, err = buf.GetUint16(); err != nil {
				return nil, err
			}
		}

	}

	if cmd.Bits&protocol.CMButtons != 0 {
		if cmd.CMButtons, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.Bits&protocol.CMImpulse != 0 {
		if cmd.CMImpulse, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	if cmd.ProtocolVersion <= 26 && cmd.Bits&protocol.CMMsec != 0 {
		if cmd.CMMsec, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	} else if cmd.ProtocolVersion >= 27 {
		if cmd.CMMsec, err = buf.ReadByte(); err != nil {
			return nil, err
		}
	}

	return &cmd, nil
}
