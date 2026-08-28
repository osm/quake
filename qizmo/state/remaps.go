package state

import "strings"

type remapTable [byteValueCount]byte

const (
	listChunkStartOffset = 1
	listChunkDataOffset  = 2
	firstPrecacheIndex   = 1
	maxListChunks        = 8
	maxRemapIndex        = byteValueCount - 1
)

func (st *Packet) RebuildRemaps() {
	if st.buildRemaps() {
		st.remapsBuilt = true
	}
}

func (st *Packet) RemapsReady() bool {
	return st.remapsBuilt
}

func (st *Packet) findPrecacheSound(name string) int {
	lo, hi := 1, len(st.precacheSounds)
	for lo < hi {
		mid := (lo + hi) / 2
		cmp := strings.Compare(st.precacheSounds[mid], name)
		if cmp < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(st.precacheSounds) && st.precacheSounds[lo] == name {
		return lo
	}
	return 0
}

func (st *Packet) findPrecacheModel(name string) int {
	if !strings.HasPrefix(name, "progs/") {
		return 0
	}
	suffix := strings.TrimPrefix(name, "progs/")
	lo, hi := 1, len(st.precacheModels)
	for lo < hi {
		mid := (lo + hi) / 2
		precache := strings.TrimPrefix(st.precacheModels[mid], "progs/")
		cmp := strings.Compare(precache, suffix)
		if cmp < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(st.precacheModels) {
		precache := strings.TrimPrefix(st.precacheModels[lo], "progs/")
		if precache == suffix {
			return lo
		}
	}
	return 0
}

func applyRemapSwap(direct, inverse *remapTable, pos int, candidate int) {
	if pos <= 0 || pos > maxRemapIndex || candidate <= 0 || candidate > maxRemapIndex {
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

func (st *Packet) buildRemaps() bool {
	if len(st.modelListChunks) == 0 && len(st.soundListChunks) == 0 {
		return false
	}

	st.soundRemap = buildListRemap(st.soundListChunks, 0, st.findPrecacheSound)
	// Qizmo carries the sound-list chunk index into the model-list pass.
	st.modelRemap = buildListRemap(st.modelListChunks, len(st.soundListChunks), st.findPrecacheModel)
	return true
}

func buildListRemap(chunks [][]byte, chunkIndex int, findPrecache func(string) int) remapTable {
	direct := identityRemap()
	inverse := direct
	if len(chunks) == 0 {
		return inverse
	}

	position := firstPrecacheIndex
	chunk := chunks[0]
	offset := listChunkDataOffset
	for position <= maxRemapIndex && offset < len(chunk) {
		if chunk[offset] == 0 {
			break
		}
		end := offset
		for end < len(chunk) && chunk[end] != 0 {
			end++
		}
		if end >= len(chunk) {
			break
		}
		if candidate := findPrecache(string(chunk[offset:end])); candidate != 0 {
			applyRemapSwap(&direct, &inverse, position, candidate)
		}

		offset = end + 1
		if offset >= len(chunk) {
			break
		}
		if chunk[offset] == 0 {
			chunkIndex++
			if chunkIndex >= len(chunks) {
				break
			}
			chunk = chunks[chunkIndex]
			offset = listChunkDataOffset
		}
		position++
	}
	return inverse
}

func identityRemap() remapTable {
	var remap remapTable
	for i := range remap {
		remap[i] = byte(i)
	}
	return remap
}

func (st *Packet) AddModelChunk(chunk []byte) bool {
	if storeListChunk(&st.modelListChunks, chunk) {
		st.remapsBuilt = false
		return true
	}
	return false
}

func (st *Packet) AddSoundChunk(chunk []byte) bool {
	if storeListChunk(&st.soundListChunks, chunk) {
		st.remapsBuilt = false
		return true
	}
	return false
}

func storeListChunk(chunks *[][]byte, chunk []byte) bool {
	if len(chunk) < listChunkDataOffset {
		return false
	}
	start := chunk[listChunkStartOffset]
	if start == 0 {
		*chunks = (*chunks)[:0]
	}
	for _, existing := range *chunks {
		if len(existing) >= listChunkDataOffset && existing[listChunkStartOffset] == start {
			return false
		}
	}
	if len(*chunks) >= maxListChunks {
		return false
	}
	buf := append([]byte(nil), chunk...)
	*chunks = append(*chunks, buf)
	return true
}
