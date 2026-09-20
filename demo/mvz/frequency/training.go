package frequency

import "encoding/binary"

// MarshalBinary writes an MVZF artifact for embedding or training.
// This layout is distinct from Qizmo's compress.dat.
func (m *Models) MarshalBinary() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	out := make([]byte, 0, modelDataSize)
	out = append(out, modelDataMagic...)
	out = binary.LittleEndian.AppendUint16(out, ModelDataFormatVersion)
	out = binary.LittleEndian.AppendUint16(out, modelDataHeaderSize)
	out = binary.LittleEndian.AppendUint32(out, uint32(ModelCount))
	out = binary.LittleEndian.AppendUint32(out, Symbols)
	for model := range m.Cumulative {
		for _, bound := range &m.Cumulative[model] {
			out = binary.LittleEndian.AppendUint32(out, bound)
		}
	}
	return out, nil
}
