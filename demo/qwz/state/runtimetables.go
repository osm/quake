package state

import (
	"strings"
)

func (st *Packet) RebuildRemaps() {
	if st.rebuildRuntimeTables() {
		st.TablesBuilt = true
	}
}

func (st *Packet) FindPrecacheSound(name string) int {
	lo, hi := 1, len(st.PrecacheSounds)
	for lo < hi {
		mid := (lo + hi) / 2
		cmp := strings.Compare(st.PrecacheSounds[mid], name)
		if cmp < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(st.PrecacheSounds) && st.PrecacheSounds[lo] == name {
		return lo
	}
	return 0
}

func (st *Packet) FindPrecacheModel(name string) int {
	if !strings.HasPrefix(name, "progs/") {
		return 0
	}
	suffix := name[6:]
	lo, hi := 1, len(st.PrecacheModels)
	for lo < hi {
		mid := (lo + hi) / 2
		precache := strings.TrimPrefix(st.PrecacheModels[mid], "progs/")
		cmp := strings.Compare(precache, suffix)
		if cmp < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(st.PrecacheModels) {
		precache := strings.TrimPrefix(st.PrecacheModels[lo], "progs/")
		if precache == suffix {
			return lo
		}
	}
	return 0
}

func applyRuntimeSwap(direct, inverse *[256]byte, pos int, candidate int) {
	if pos <= 0 || pos > 0xff || candidate <= 0 || candidate > 0xff {
		return
	}
	if int(direct[pos]) == candidate {
		return
	}
	old := direct[pos]
	target := byte(candidate)
	if int(direct[candidate]) != candidate {
		target = inverse[candidate]
	}
	direct[pos] = byte(candidate)
	inverse[candidate] = byte(pos)
	direct[target] = old
	inverse[old] = target
}

func (st *Packet) rebuildRuntimeTables() bool {
	if len(st.ModelChunks) == 0 && len(st.SoundChunks) == 0 {
		return false
	}
	var directFreq, invFreq, directSound, invSound [256]byte
	for i := 0; i < 256; i++ {
		b := byte(i)
		directFreq[i] = b
		invFreq[i] = b
		directSound[i] = b
		invSound[i] = b
	}

	pos := 1
	chunkIdx := 0
	var chunk []byte
	if len(st.SoundChunks) != 0 {
		chunk = st.SoundChunks[0]
	}
	i := 2
	for pos <= 0xff && chunk != nil && i < len(chunk) {
		if chunk[i] == 0 {
			break
		}
		j := i
		for j < len(chunk) && chunk[j] != 0 {
			j++
		}
		if j >= len(chunk) {
			break
		}
		name := string(chunk[i:j])
		if candidate := st.FindPrecacheSound(name); candidate != 0 {
			applyRuntimeSwap(&directSound, &invSound, pos, candidate)
		}
		i = j + 1
		if i >= len(chunk) {
			break
		}
		if chunk[i] == 0 {
			chunkIdx++
			if chunkIdx >= len(st.SoundChunks) {
				break
			}
			chunk = st.SoundChunks[chunkIdx]
			i = 2
		}
		pos++
	}

	pos = 1
	chunkIdx = len(st.SoundChunks)
	chunk = nil
	if len(st.ModelChunks) != 0 {
		chunk = st.ModelChunks[0]
	}
	i = 2
	for pos <= 0xff && chunk != nil && i < len(chunk) {
		if chunk[i] == 0 {
			break
		}
		j := i
		for j < len(chunk) && chunk[j] != 0 {
			j++
		}
		if j >= len(chunk) {
			break
		}
		name := string(chunk[i:j])
		if candidate := st.FindPrecacheModel(name); candidate != 0 {
			applyRuntimeSwap(&directFreq, &invFreq, pos, candidate)
		}
		i = j + 1
		if i >= len(chunk) {
			break
		}
		if chunk[i] == 0 {
			chunkIdx++
			if chunkIdx >= len(st.ModelChunks) {
				break
			}
			chunk = st.ModelChunks[chunkIdx]
			i = 2
		}
		pos++
	}

	st.ModelRemap = invFreq
	st.SoundRemap = invSound
	return true
}

func (st *Packet) AddModelChunk(chunk []byte) bool {
	if storeListChunk(&st.ModelChunks, chunk) {
		st.TablesBuilt = false
		return true
	}
	return false
}

func (st *Packet) AddSoundChunk(chunk []byte) bool {
	if storeListChunk(&st.SoundChunks, chunk) {
		st.TablesBuilt = false
		return true
	}
	return false
}

func storeListChunk(chunks *[][]byte, chunk []byte) bool {
	if len(chunk) < 2 {
		return false
	}
	start := chunk[1]
	if start == 0 {
		*chunks = (*chunks)[:0]
	}
	for _, existing := range *chunks {
		if len(existing) >= 2 && existing[1] == start {
			return false
		}
	}
	if len(*chunks) >= 8 {
		return false
	}
	buf := append([]byte(nil), chunk...)
	*chunks = append(*chunks, buf)
	return true
}
