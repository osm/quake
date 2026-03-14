package standard

import (
	"encoding/binary"

	"github.com/osm/quake/demo/qwz/state"
)

func (d *decoder) parseSpawnBaseline() error {
	entNum, err := d.r.ReadU16()
	if err != nil {
		return err
	}
	base, err := d.r.ReadN(13)
	if err != nil {
		return err
	}

	var rec state.EntityRecord
	state.SetEntityNumber(&rec, entNum)
	state.SetEntityRecordByte(&rec, 4, base[0])
	state.SetEntityRecordByte(&rec, 5, base[1])
	state.SetEntityRecordByte(&rec, 6, base[2])
	state.SetEntityRecordByte(&rec, 7, base[3])
	state.SetEntityRecordByte(&rec, 8, 0)
	state.SetEntityRecordU16(&rec, 12, binary.LittleEndian.Uint16(base[4:6]))
	state.SetEntityRecordByte(&rec, 9, base[6])
	state.SetEntityRecordU16(&rec, 14, binary.LittleEndian.Uint16(base[7:9]))
	state.SetEntityRecordByte(&rec, 10, base[9])
	state.SetEntityRecordU16(&rec, 16, binary.LittleEndian.Uint16(base[10:12]))
	state.SetEntityRecordByte(&rec, 11, base[12])

	d.st.Baselines[entNum] = rec
	d.st.EntityRaw[entNum] = rec
	d.st.EntityLast[entNum] = rec
	d.st.EntityLastRaw[entNum] = true
	return nil
}
