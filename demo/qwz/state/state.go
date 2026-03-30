package state

import (
	"fmt"
)

type Packet struct {
	PlayerIndex     byte
	SeqNoRel        uint32
	PlayerFreqIndex byte
	CmdSeqNo        uint32

	Scale64       [64]byte
	Tag64         [64]uint32
	ModelRemap    [256]byte
	SoundRemap    [256]byte
	PlayerHistory [32]PlayerHistory

	CurrentPlayers []PlayerRecord

	Baselines      map[uint16]EntityRecord
	EntityLast     map[uint16]EntityRecord
	EntityRaw      map[uint16]EntityRecord
	EntityLastRaw  map[uint16]bool
	EntityHistory  [32]EntityHistory
	RawEntByteHist [256]map[uint16]EntityRecord

	CenterPrintStrings map[uint16][]byte
	PrintStrings       map[uint16][]byte
	PrintMode3Strings  map[uint16][]byte
	StuffTextStrings   map[uint16][]byte
	SetInfoStrings     map[uint16][]byte

	ModelNames     map[int]string
	SoundNames     map[int]string
	PrecacheModels []string
	PrecacheSounds []string
	ModelChunks    [][]byte
	SoundChunks    [][]byte

	PacketBaseSeq       uint32
	PacketHasBase       bool
	PacketEntsCommitted bool
	TableVariant        string
	TablesBuilt         bool
}

func NewPacket(playerIndex byte) *Packet {
	st := &Packet{
		PlayerIndex:   playerIndex,
		Baselines:     make(map[uint16]EntityRecord),
		EntityLast:    make(map[uint16]EntityRecord),
		EntityRaw:     make(map[uint16]EntityRecord),
		EntityLastRaw: make(map[uint16]bool),
		ModelNames:    make(map[int]string),
		SoundNames:    make(map[int]string),
	}

	for i := 0; i < 64; i++ {
		st.Scale64[i] = 1
	}

	for i := 0; i < 256; i++ {
		st.ModelRemap[i] = byte(i)
		st.SoundRemap[i] = byte(i)
	}

	return st
}

func (st *Packet) SeqNo() uint32 {
	return st.SeqNoRel
}

func (st *Packet) NumCurrentPlayers() int {
	return len(st.CurrentPlayers)
}

func (st *Packet) FirstCurrentPlayer() PlayerRecord {
	if len(st.CurrentPlayers) == 0 {
		return PlayerRecord{}
	}

	return st.CurrentPlayers[0]
}

func (st *Packet) LastCurrentPlayer() PlayerRecord {
	if len(st.CurrentPlayers) == 0 {
		return PlayerRecord{}
	}

	return st.CurrentPlayers[len(st.CurrentPlayers)-1]
}

func (st *Packet) Scale(seq uint32) byte {
	return st.Scale64[seq&0x3f]
}

func (st *Packet) PlayerHistoryEntry(seq uint32) (PlayerHistory, bool) {
	h := st.PlayerHistory[seq&0x1f]

	if !h.Valid || h.Seq != seq {
		return PlayerHistory{}, false
	}

	return h, true
}

func (st *Packet) ModelRemapByte(idx byte) byte {
	return st.ModelRemap[idx]
}

func (st *Packet) SoundRemapByte(idx byte) byte {
	return st.SoundRemap[idx]
}

func (st *Packet) BeginPacket(seq uint32) {
	st.SeqNoRel = seq & 0x7fffffff
	st.CurrentPlayers = st.CurrentPlayers[:0]
	st.PacketBaseSeq = 0
	st.PacketHasBase = false
	st.PacketEntsCommitted = false
}

func (st *Packet) CommitCmdScale(msec byte) {
	st.CmdSeqNo++
	st.Tag64[st.CmdSeqNo&0x3f] = st.CmdSeqNo
	st.Scale64[st.CmdSeqNo&0x3f] = msec
}

func (st *Packet) CommitPacketAs(seq uint32) {
	s32 := &st.PlayerHistory[seq&0x1f]
	s32.Seq = seq
	s32.Valid = true
	s32.Recs = append(s32.Recs[:0], st.CurrentPlayers...)
}

func (st *Packet) CommitPacket() {
	if !st.PacketEntsCommitted && st.PacketHasBase {
		if baseEnts, ok := st.FindEntHistoryMapBySeq(st.PacketBaseSeq); ok {
			st.CommitEntities(st.SeqNoRel, baseEnts)
		}
	}
	st.CommitPacketAs(st.SeqNoRel)
}

func (st *Packet) PlayerName(idx uint16) []byte {
	// Chat-symbol name expansion is not needed for byte-identical .qwd
	// output, so this remains a placeholder until a sample requires real
	// name tracking.
	// A full implementation would need to capture player names from
	// svc_updateuserinfo and svc_setinfo and keep them in shared decoder
	// state.
	return []byte("unnamed")
}

func (st *Packet) DecodeChatSym(sym uint32, mode3 bool) ([]byte, error) {
	if sym < 0x100 {
		return []byte{byte(sym)}, nil
	}

	name := st.PlayerName(uint16(sym & 0x1f))

	if mode3 {
		if sym < 0x120 {
			out := make([]byte, 0, len(name)+4)
			out = append(out, '(')
			out = append(out, name...)
			out = append(out, ')', ':', ' ')
			return out, nil
		}
		if sym < 0x140 {
			return append([]byte(nil), name...), nil
		}
		if s, ok := st.PrintMode3Strings[uint16(sym)]; ok {
			return append([]byte(nil), s...), nil
		}
	} else {
		if sym < 0x120 {
			return append([]byte(nil), name...), nil
		}
		if s, ok := st.PrintStrings[uint16(sym)]; ok {
			return append([]byte(nil), s...), nil
		}
	}

	return nil, fmt.Errorf(
		"seq=%d op 0x08 missing string for sym %#x mode3=%v",
		st.SeqNoRel,
		sym,
		mode3,
	)
}

func (st *Packet) CommitRawEntitiesByte(seqByte byte, ents map[uint16]EntityRecord) {
	st.RawEntByteHist[seqByte] = CloneEntMap(ents)
}

func (st *Packet) FindRawEntitiesByte(seqByte byte) (map[uint16]EntityRecord, bool) {
	ents := st.RawEntByteHist[seqByte]
	if ents == nil {
		return nil, false
	}
	return ents, true
}

func (st *Packet) CommitEntities(seq uint32, ents map[uint16]EntityRecord) {
	slot := &st.EntityHistory[seq&0x1f]
	slot.Seq = seq
	slot.Valid = true
	slot.Ents = CloneEntMap(ents)
	slot.Ordered = sortedEntitySlice(slot.Ents)
}

func (st *Packet) ResetEntityTracking() {
	clear(st.Baselines)
	clear(st.EntityLast)
	clear(st.EntityRaw)
	clear(st.EntityLastRaw)

	for i := range st.EntityHistory {
		st.EntityHistory[i] = EntityHistory{}
	}

	for i := range st.RawEntByteHist {
		st.RawEntByteHist[i] = nil
	}

	st.PacketBaseSeq = 0
	st.PacketHasBase = false
	st.PacketEntsCommitted = false
}

func (st *Packet) DefaultRec() PlayerRecord {
	var r PlayerRecord
	SetPlayerRecordByte(&r, 0x0b, st.PlayerIndex)
	SetPlayerRecordByte(&r, 0x19, st.PlayerFreqIndex)
	return r
}
