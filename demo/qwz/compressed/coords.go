package compressed

import "github.com/osm/quake/demo/qwz/state"

func resolvePlayerCoords(
	players []state.PlayerRecord,
	playerID uint16,
) (uint16, uint16, uint16, bool) {
	for i := range players {
		if state.PlayerRecordByte(players[i], 0x0b) == byte(playerID-1) {
			return uint16(players[i][1] & 0xffff),
				uint16((players[i][1] >> 16) & 0xffff),
				uint16(players[i][2] & 0xffff),
				true
		}
	}

	return 0, 0, 0, false
}

func resolveEntityCoords(
	st *state.Packet,
	entityID uint16,
) (uint16, uint16, uint16, bool, bool) {
	if ents, ok := st.FindEntHistoryBySeq(st.SeqNo()); ok {
		found := false
		stoppedBefore := false

		for _, rec := range ents {
			ent := state.EntityNumber(rec)
			if ent == entityID {
				return state.EntityRecordU16(rec, 12),
					state.EntityRecordU16(rec, 14),
					state.EntityRecordU16(rec, 16),
					true,
					false
			}

			if entityID < ent {
				stoppedBefore = true
				break
			}
		}

		return 0, 0, 0, found, stoppedBefore
	}

	return 0, 0, 0, false, false
}
