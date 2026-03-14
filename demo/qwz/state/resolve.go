package state

import (
	"fmt"
	"sort"
)

func (st *Packet) FindBaseline(refSeq uint32) (PlayerRecord, bool, error) {
	if st.Tag64[refSeq&0x3f] != refSeq {
		return PlayerRecord{}, false, fmt.Errorf(
			"player history tag mismatch for refSeq=%d",
			refSeq,
		)
	}
	h := st.PlayerHistory[refSeq&0x1f]
	if !h.Valid || h.Seq != refSeq {
		return PlayerRecord{}, false, fmt.Errorf(
			"player history slot mismatch for refSeq=%d",
			refSeq,
		)
	}
	for _, rec := range h.Recs {
		if PlayerRecordByte(rec, 0x0b) == st.PlayerIndex {
			return rec, true, nil
		}
	}
	return PlayerRecord{}, false, nil
}

func (st *Packet) FindEntHistoryBySeq(seq uint32) ([]EntityRecord, bool) {
	h := &st.EntityHistory[seq&0x1f]
	if !h.Valid || h.Seq != seq {
		return nil, false
	}
	return h.Ordered, true
}

func (st *Packet) FindEntHistoryMapBySeq(seq uint32) (map[uint16]EntityRecord, bool) {
	h := &st.EntityHistory[seq&0x1f]
	if !h.Valid || h.Seq != seq {
		return nil, false
	}
	return h.Ents, true
}

func CloneEntMap(src map[uint16]EntityRecord) map[uint16]EntityRecord {
	dst := make(map[uint16]EntityRecord, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func sortedEntitySlice(src map[uint16]EntityRecord) []EntityRecord {
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
