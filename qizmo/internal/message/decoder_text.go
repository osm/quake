package message

import (
	"fmt"

	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/freq"
	"github.com/osm/quake/qizmo/state"
)

func (d *packetDecoder) decodeSVCPrint(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	mode, err := rd.DecodeFreqByte(ft, freq.SVCPrintMode)
	if err != nil {
		return nil, err
	}
	out = append(out, mode)

	chat := mode == protocol.PrintChat
	model := uint32(freq.SVCPrintString)
	if chat {
		model = freq.SVCPrintChatString
	}

	for {
		sym, err := rd.DecodeFreqSymbol(ft, model, freq.PairedSymbols)
		if err != nil {
			return nil, err
		}
		s, err := decodePrintSymbol(st, sym, chat)
		if err != nil {
			return nil, err
		}
		out = append(out, s...)
		if sym == 0 {
			return out, nil
		}
	}
}

func decodePrintSymbol(st *state.Packet, symbol uint32, chat bool) ([]byte, error) {
	if symbol < freq.Symbols {
		return []byte{byte(symbol)}, nil
	}

	name := st.PlayerName(byte(symbol & (maxPlayers - 1)))
	if chat {
		if symbol < printDictionaryStart {
			out := make([]byte, 0, len(name)+4)
			out = append(out, '(')
			out = append(out, name...)
			return append(out, ')', ':', ' '), nil
		}
		if symbol < chatDictionaryStart {
			return append([]byte(nil), name...), nil
		}
		if text, ok := st.PrintChatStrings[uint16(symbol)]; ok {
			return append([]byte(nil), text...), nil
		}
	} else {
		if symbol < printDictionaryStart {
			return append([]byte(nil), name...), nil
		}
		if text, ok := st.PrintStrings[uint16(symbol)]; ok {
			return append([]byte(nil), text...), nil
		}
	}

	return nil, fmt.Errorf(
		"seq=%d svc_print missing string for sym %#x chat=%v",
		st.Sequence(),
		symbol,
		chat,
	)
}

func (d *packetDecoder) decodeSVCStuffText(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	for {
		sym, err := rd.DecodeFreqSymbol(ft, freq.SVCStuffText, freq.PairedSymbols)
		if err != nil {
			return nil, err
		}
		if sym < freq.Symbols {
			out = append(out, byte(sym))
		} else {
			s, ok := st.StuffTextStrings[uint16(sym)]
			if !ok {
				return nil, fmt.Errorf(
					"seq=%d svc_stufftext missing string for sym %#x",
					st.Sequence(),
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

func (d *packetDecoder) decodeSVCCenterPrint(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state
	for {
		sym, err := rd.DecodeFreqSymbol(ft, freq.SVCCenterPrintString, freq.PairedSymbols)
		if err != nil {
			return nil, err
		}
		if sym < freq.Symbols {
			out = append(out, byte(sym))
		} else {
			s, ok := st.CenterPrintStrings[uint16(sym)]
			if !ok {
				return nil, fmt.Errorf(
					"seq=%d svc_centerprint missing string for sym %#x",
					st.Sequence(),
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

func (d *packetDecoder) decodeSVCSetInfo(out []byte) ([]byte, error) {
	rd := d.rd
	ft := d.ft
	st := d.state

	payloadStart := len(out)
	slot, err := rd.DecodeFreqByte(ft, freq.SVCPlayerIndex)
	if err != nil {
		return nil, err
	}
	out = append(out, slot)

	for {
		sym, err := rd.DecodeFreqSymbol(ft, freq.SVCSetInfoKey, freq.PairedSymbols)
		if err != nil {
			return nil, err
		}
		if sym < freq.Symbols {
			out = append(out, byte(sym))
		} else {
			s, ok := st.SetInfoStrings[uint16(sym)]
			if !ok {
				return nil, fmt.Errorf(
					"seq=%d svc_setinfo missing string for sym %#x",
					st.Sequence(),
					sym,
				)
			}
			out = append(out, s...)
		}
		if sym == 0 {
			break
		}
	}

	out, err = d.decodeString(out, freq.SVCSetInfoValue)
	if err != nil {
		return nil, err
	}
	trackPlayerIdentity(st, protocol.SVCSetInfo, out[payloadStart:])
	return out, nil
}

func (d *packetDecoder) decodeSVCUpdateUserInfo(out []byte) ([]byte, error) {
	payloadStart := len(out)
	var err error
	out, err = d.appendFreqBytes(out, freq.SVCPlayerIndex, 1)
	if err != nil {
		return nil, err
	}
	out, err = d.appendFreqBytes(out, freq.SVCUpdateUserInfoUserID, 4)
	if err != nil {
		return nil, err
	}
	out, err = d.decodeString(out, freq.SVCUpdateUserInfoString)
	if err != nil {
		return nil, err
	}
	trackPlayerIdentity(d.state, protocol.SVCUpdateUserInfo, out[payloadStart:])
	return out, nil
}

func (d *packetDecoder) decodeSVCLightStyle(out []byte) ([]byte, error) {
	var err error
	out, err = d.appendFreqBytes(out, freq.ByteValue, 1)
	if err != nil {
		return nil, err
	}
	return d.decodeString(out, freq.ByteValue)
}

func (d *packetDecoder) decodeSVCServerInfo(out []byte) ([]byte, error) {
	var err error
	out, err = d.decodeString(out, freq.SVCServerInfoString)
	if err != nil {
		return nil, err
	}
	return d.decodeString(out, freq.SVCServerInfoString)
}

func (d *packetDecoder) decodeString(
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
