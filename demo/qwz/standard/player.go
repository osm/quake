package standard

import (
	"encoding/binary"

	"github.com/osm/quake/demo/qwz/state"
)

func (d *decoder) parsePlayerInfo(seq uint32) error {
	player, err := d.r.ReadByte()
	if err != nil {
		return err
	}
	var flags uint16
	if err := binary.Read(d.r, binary.LittleEndian, &flags); err != nil {
		return err
	}
	posXY, err := d.r.ReadN(4)
	if err != nil {
		return err
	}
	posTail, err := d.r.ReadN(3)
	if err != nil {
		return err
	}

	rec := d.st.DefaultRec()
	state.SetPlayerRecordByte(&rec, 0x0b, player)
	rec[1] = binary.LittleEndian.Uint32(posXY)
	state.SetPlayerRecordByte(&rec, 0x08, posTail[0])
	state.SetPlayerRecordByte(&rec, 0x09, posTail[1])
	state.SetPlayerRecordByte(&rec, 0x0a, posTail[2])

	if flags&0x0001 != 0 {
		v, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		state.SetPlayerRecordByte(&rec, 0x2e, v)
	}

	if flags&0x0002 != 0 {
		extra, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		if extra&0x01 != 0 {
			v, err := d.r.ReadU16()
			if err != nil {
				return err
			}
			recB := state.PlayerRecordBytesLE(rec)
			state.SetU16LE(&recB, 14, v)
			rec = state.PlayerRecordFromBytesLE(recB)
		}
		if extra&0x02 != 0 {
			if _, err := d.r.ReadU16(); err != nil {
				return err
			}
		}
		if extra&0x80 != 0 {
			v, err := d.r.ReadU16()
			if err != nil {
				return err
			}
			recB := state.PlayerRecordBytesLE(rec)
			state.SetU16LE(&recB, 12, v)
			rec = state.PlayerRecordFromBytesLE(recB)
		}
		if extra&0x04 != 0 {
			v, err := d.r.ReadU16()
			if err != nil {
				return err
			}
			recB := state.PlayerRecordBytesLE(rec)
			state.SetU16LE(&recB, 20, v)
			rec = state.PlayerRecordFromBytesLE(recB)
		}
		if extra&0x08 != 0 {
			v, err := d.r.ReadU16()
			if err != nil {
				return err
			}
			recB := state.PlayerRecordBytesLE(rec)
			state.SetU16LE(&recB, 22, v)
			rec = state.PlayerRecordFromBytesLE(recB)
		}
		if extra&0x10 != 0 {
			v, err := d.r.ReadU16()
			if err != nil {
				return err
			}
			recB := state.PlayerRecordBytesLE(rec)
			state.SetU16LE(&recB, 18, v)
			rec = state.PlayerRecordFromBytesLE(recB)
		}
		if extra&0x40 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetPlayerRecordByte(&rec, 0x10, v)
		}
		v, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		state.SetPlayerRecordByte(&rec, 0x18, v)
	}

	if flags&0x0004 != 0 {
		v, err := d.r.ReadU16()
		if err != nil {
			return err
		}
		rec[7] = (rec[7] & 0xffff0000) | uint32(v)
	}
	if flags&0x0008 != 0 {
		v, err := d.r.ReadU16()
		if err != nil {
			return err
		}
		rec[7] = (rec[7] & 0x0000ffff) | (uint32(v) << 16)
	}
	if flags&0x0010 != 0 {
		v, err := d.r.ReadU16()
		if err != nil {
			return err
		}
		rec[8] = (rec[8] & 0xffff0000) | uint32(v)
	}
	if flags&0x0020 != 0 {
		v, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		state.SetPlayerRecordByte(&rec, 0x19, v)
	}
	if flags&0x0040 != 0 {
		v, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		state.SetPlayerRecordByte(&rec, 0x1a, v)
	}
	if flags&0x0080 != 0 {
		v, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		state.SetPlayerRecordByte(&rec, 0x1b, v)
	}
	if flags&0x0100 != 0 {
		v, err := d.r.ReadByte()
		if err != nil {
			return err
		}
		state.SetPlayerRecordByte(&rec, 0x22, v)
	}
	d.st.CurrentPlayers = append(d.st.CurrentPlayers, rec)
	return nil
}
