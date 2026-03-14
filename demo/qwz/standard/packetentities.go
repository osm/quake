package standard

import "github.com/osm/quake/demo/qwz/state"

func (d *decoder) parsePacketEntities(base map[uint16]state.EntityRecord) error {
	ents := state.CloneEntMap(base)
	for {
		hdr, err := d.r.ReadU16()
		if err != nil {
			return err
		}
		if hdr == 0 {
			d.packetEnts = ents
			return nil
		}

		entNum := hdr & 0x01ff
		bits := hdr & 0xfe00
		if bits&0x8000 != 0 {
			lo, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			bits |= uint16(lo)
		}
		if bits&0x4000 != 0 {
			delete(ents, entNum)
			continue
		}

		rec, ok := ents[entNum]
		if !ok {
			rec = d.st.Baselines[entNum]
			state.SetEntityNumber(&rec, entNum)
		}

		if bits&0x0004 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetEntityRecordByte(&rec, 4, v)
		}
		if bits&0x2000 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetEntityRecordByte(&rec, 5, v)
		}
		if bits&0x0008 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetEntityRecordByte(&rec, 6, v)
		}
		if bits&0x0010 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetEntityRecordByte(&rec, 7, v)
		}
		if bits&0x0020 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetEntityRecordByte(&rec, 8, v)
		}
		if bits&0x0200 != 0 {
			v, err := d.r.ReadU16()
			if err != nil {
				return err
			}
			state.SetEntityRecordU16(&rec, 12, v)
		}
		if bits&0x0001 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetEntityRecordByte(&rec, 9, v)
		}
		if bits&0x0400 != 0 {
			v, err := d.r.ReadU16()
			if err != nil {
				return err
			}
			state.SetEntityRecordU16(&rec, 14, v)
		}
		if bits&0x1000 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetEntityRecordByte(&rec, 10, v)
		}
		if bits&0x0800 != 0 {
			v, err := d.r.ReadU16()
			if err != nil {
				return err
			}
			state.SetEntityRecordU16(&rec, 16, v)
		}
		if bits&0x0002 != 0 {
			v, err := d.r.ReadByte()
			if err != nil {
				return err
			}
			state.SetEntityRecordByte(&rec, 11, v)
		}

		ents[entNum] = rec
	}
}
