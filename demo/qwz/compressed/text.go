package compressed

import (
	"fmt"
	"github.com/osm/quake/demo/qwz/freq"
)

func (d *decoder) decodeSVCPrint(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	mode, err := rd.DecodeFreqByte(ft, freq.SVCPrintMode)
	if err != nil {
		return nil, err
	}
	out = append(out, mode)

	mode3 := mode == 3
	model := uint32(freq.SVCPrintString)
	if mode3 {
		model = freq.SVCPrintString3
	}

	for {
		sym, err := rd.DecodeFreqSymbol(ft, model, 0x200)
		if err != nil {
			return nil, err
		}
		s, err := st.DecodeChatSym(sym, mode3)
		if err != nil {
			return nil, err
		}
		out = append(out, s...)
		if sym == 0 {
			return out, nil
		}
	}
}

func (d *decoder) decodeSVCStufftext(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	for {
		sym, err := rd.DecodeFreqSymbol(ft, freq.SVCStufftext, 0x200)
		if err != nil {
			return nil, err
		}
		if sym < 0x100 {
			out = append(out, byte(sym))
		} else {
			s, ok := st.StuffTextStrings[uint16(sym)]
			if !ok {
				return nil, fmt.Errorf(
					"seq=%d op 0x09 missing string for sym %#x",
					st.SeqNo(),
					sym,
				)
			}
			out = append(out, s...)
		}
		if sym == 0 {
			return out, nil
		}
	}
}

func (d *decoder) decodeSVCCenterPrint(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	for {
		sym, err := rd.DecodeFreqSymbol(ft, freq.SVCCenterPrintString, 0x200)
		if err != nil {
			return nil, err
		}
		if sym < 0x100 {
			out = append(out, byte(sym))
		} else {
			s, ok := st.CenterPrintStrings[uint16(sym)]
			if !ok {
				return nil, fmt.Errorf(
					"seq=%d op 0x1a missing string for sym %#x",
					st.SeqNo(),
					sym,
				)
			}
			out = append(out, s...)
		}
		if sym == 0 {
			return out, nil
		}
	}
}

func (d *decoder) decodeSVCSetInfo(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state

	slot, err := rd.DecodeFreqByte(ft, freq.PlayerInfoSlot1)
	if err != nil {
		return nil, err
	}
	out = append(out, slot)

	for {
		sym, err := rd.DecodeFreqSymbol(ft, freq.SVCSetInfoKey, 0x200)
		if err != nil {
			return nil, err
		}
		if sym < 0x100 {
			out = append(out, byte(sym))
		} else {
			s, ok := st.SetInfoStrings[uint16(sym)]
			if !ok {
				return nil, fmt.Errorf(
					"seq=%d op 0x33 missing string for sym %#x",
					st.SeqNo(),
					sym,
				)
			}
			out = append(out, s...)
		}
		if sym == 0 {
			break
		}
	}

	return d.decodeString(out, freq.SVCSetInfoValue)
}

func (d *decoder) decodeString(
	out []byte,
	freqTableAddr uint32,
) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	for {
		b, err := rd.DecodeFreqByte(ft, freqTableAddr)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
		if b == 0 {
			return out, nil
		}
	}
}
