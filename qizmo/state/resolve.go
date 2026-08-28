package state

import (
	"fmt"
	"sort"
)

func (st *Packet) FindPlayerBaseline(referenceSequence uint32) (PlayerRecord, bool, error) {
	if !st.HasCommandScale(referenceSequence) {
		return PlayerRecord{}, false, fmt.Errorf(
			"player history tag mismatch for reference sequence %d",
			referenceSequence,
		)
	}
	history := st.playerHistory[referenceSequence&packetHistoryMask]
	if !history.valid || history.sequence != referenceSequence {
		return PlayerRecord{}, false, fmt.Errorf(
			"player history slot mismatch for reference sequence %d",
			referenceSequence,
		)
	}
	for _, rec := range history.players {
		if PlayerIndex(rec) == st.PlayerIndex {
			return rec, true, nil
		}
	}
	return PlayerRecord{}, false, nil
}

func (st *Packet) EntitySnapshot(sequence uint32) ([]EntityRecord, bool) {
	history := &st.entityHistory[sequence&packetHistoryMask]
	if !history.valid || history.sequence != sequence {
		return nil, false
	}
	return history.ordered, true
}

func (st *Packet) EntitySnapshotMap(sequence uint32) (map[uint16]EntityRecord, bool) {
	history := &st.entityHistory[sequence&packetHistoryMask]
	if !history.valid || history.sequence != sequence {
		return nil, false
	}
	return history.entities, true
}

func CloneEntityMap(src map[uint16]EntityRecord) map[uint16]EntityRecord {
	dst := make(map[uint16]EntityRecord, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func sortedEntityRecords(src map[uint16]EntityRecord) []EntityRecord {
	out := make([]EntityRecord, 0, len(src))
	keys := make([]uint16, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		out = append(out, src[k])
	}
	return out
}
