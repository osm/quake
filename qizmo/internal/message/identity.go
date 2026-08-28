package message

import (
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/state"
)

func trackPlayerIdentity(st *state.Packet, opcode byte, payload []byte) {
	switch opcode {
	case protocol.SVCUpdateUserInfo:
		userInfoEnd, ok := endCString(payload, updateUserInfoStringOffset)
		if !ok {
			return
		}
		st.SetPlayerUserInfo(payload[0], string(payload[updateUserInfoStringOffset:userInfoEnd-1]))

	case protocol.SVCSetInfo:
		keyEnd, ok := endCString(payload, setInfoKeyOffset)
		if !ok {
			return
		}
		valueEnd, ok := endCString(payload, keyEnd)
		if !ok || string(payload[setInfoKeyOffset:keyEnd-1]) != "name" {
			return
		}
		st.SetPlayerName(payload[0], string(payload[keyEnd:valueEnd-1]))
	}
}
