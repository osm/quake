package state

import (
	"github.com/osm/quake/common/infostring"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/qizmo/assets"
)

const (
	commandScaleHistorySize = 64
	packetHistorySize       = 32
	byteValueCount          = 1 << 8

	commandScaleHistoryMask = commandScaleHistorySize - 1
	packetHistoryMask       = packetHistorySize - 1
)

type Packet struct {
	// Connection and command state.
	PlayerIndex               byte
	PlayerModelIndex          byte
	CommandSequence           uint32
	historyValidAfterSequence uint32

	// Runtime remapping tables.
	commandScales         [commandScaleHistorySize]byte
	commandScaleSequences [commandScaleHistorySize]uint32
	modelRemap            remapTable
	soundRemap            remapTable
	remapsBuilt           bool

	// Player identity and history.
	playerHistory  [packetHistorySize]playerHistory
	playerNames    [protocol.QWMaxClients]string
	CurrentPlayers []PlayerRecord

	// Entity state and history.
	Baselines        map[uint16]EntityRecord
	entityHistory    [packetHistorySize]entityHistory
	rawEntityHistory [byteValueCount]map[uint16]EntityRecord

	// Static string dictionaries.
	CenterPrintStrings map[uint16][]byte
	PrintStrings       map[uint16][]byte
	PrintChatStrings   map[uint16][]byte
	StuffTextStrings   map[uint16][]byte
	SetInfoStrings     map[uint16][]byte

	// Precache state.
	precacheModels  []string
	precacheSounds  []string
	modelListChunks [][]byte
	soundListChunks [][]byte

	// Current packet.
	packetBaseSequence      uint32
	hasPacketBase           bool
	packetSequence          uint32
	packetEntitiesCommitted bool
}

func NewPacket(playerIndex byte) *Packet {
	st := &Packet{
		PlayerIndex: playerIndex,
		Baselines:   make(map[uint16]EntityRecord),
	}

	for i := range st.commandScales {
		st.commandScales[i] = 1
	}

	for i := range st.modelRemap {
		st.modelRemap[i] = byte(i)
		st.soundRemap[i] = byte(i)
	}

	return st
}

func NewPacketWithAssets(playerIndex byte, packetAssets assets.Assets) *Packet {
	st := NewPacket(playerIndex)
	st.CenterPrintStrings = packetAssets.CenterPrintStrings
	st.PrintStrings = packetAssets.PrintStrings
	st.PrintChatStrings = packetAssets.PrintChatStrings
	st.StuffTextStrings = packetAssets.StuffTextStrings
	st.SetInfoStrings = packetAssets.SetInfoStrings
	st.precacheModels = packetAssets.PrecacheModels
	st.precacheSounds = packetAssets.PrecacheSounds
	return st
}

func (st *Packet) Sequence() uint32 {
	return st.packetSequence
}

func (st *Packet) CommandScale(sequence uint32) byte {
	return st.commandScales[sequence&commandScaleHistoryMask]
}

func (st *Packet) HasCommandScale(sequence uint32) bool {
	return st.commandScaleSequences[sequence&commandScaleHistoryMask] == sequence
}

func (st *Packet) SetCommandScale(sequence uint32, msec byte) {
	index := sequence & commandScaleHistoryMask
	st.commandScaleSequences[index] = sequence
	st.commandScales[index] = msec
}

func (st *Packet) PlayerSnapshot(sequence uint32) ([]PlayerRecord, bool) {
	history := st.playerHistory[sequence&packetHistoryMask]

	if !history.valid || history.sequence != sequence {
		return nil, false
	}

	return history.players, true
}

func (st *Packet) ModelForRemapIndex(index byte) byte {
	return st.modelRemap[index]
}

func (st *Packet) SoundForRemapIndex(index byte) byte {
	return st.soundRemap[index]
}

func (st *Packet) ModelRemapIndex(modelIndex byte) (byte, bool) {
	return remapIndex(&st.modelRemap, modelIndex)
}

func (st *Packet) SoundRemapIndex(soundIndex byte) (byte, bool) {
	return remapIndex(&st.soundRemap, soundIndex)
}

func remapIndex(remap *remapTable, value byte) (byte, bool) {
	for i, mapped := range remap {
		if mapped == value {
			return byte(i), true
		}
	}
	return 0, false
}

func (st *Packet) BeginPacket(sequence uint32) {
	st.packetSequence = sequence & protocol.QWSequenceMask
	st.CurrentPlayers = st.CurrentPlayers[:0]
	st.packetBaseSequence = 0
	st.hasPacketBase = false
	st.packetEntitiesCommitted = false
}

func (st *Packet) SetPacketBase(sequence uint32) {
	st.packetBaseSequence = sequence
	st.hasPacketBase = true
}

func (st *Packet) PacketBase() (uint32, bool) {
	return st.packetBaseSequence, st.hasPacketBase
}

func (st *Packet) InvalidateHistory(sequence uint32) {
	st.historyValidAfterSequence = (sequence & protocol.QWSequenceMask) + uint32(len(st.playerHistory))
}

func (st *Packet) AllowsHistoryReference(sequence uint32) bool {
	return sequence > st.historyValidAfterSequence
}

func (st *Packet) CommitCommandScale(msec byte) {
	st.CommandSequence++
	st.SetCommandScale(st.CommandSequence, msec)
}

func (st *Packet) CommitPlayerSnapshot(sequence uint32) {
	history := &st.playerHistory[sequence&packetHistoryMask]
	history.sequence = sequence
	history.valid = true
	history.players = append(history.players[:0], st.CurrentPlayers...)
}

func (st *Packet) CommitPacket() {
	if !st.packetEntitiesCommitted && st.hasPacketBase {
		if baseEntities, ok := st.EntitySnapshotMap(st.packetBaseSequence); ok {
			st.CommitEntitySnapshot(st.packetSequence, baseEntities)
		}
	}
	st.CommitPlayerSnapshot(st.packetSequence)
}

func (st *Packet) PlayerName(playerIndex byte) []byte {
	if int(playerIndex) < len(st.playerNames) && st.playerNames[playerIndex] != "" {
		return []byte(st.playerNames[playerIndex])
	}
	return []byte("unnamed")
}

func (st *Packet) SetPlayerName(playerIndex byte, name string) {
	if int(playerIndex) < len(st.playerNames) {
		st.playerNames[playerIndex] = name
	}
}

func (st *Packet) PlayerNames() [protocol.QWMaxClients]string {
	return st.playerNames
}

func (st *Packet) SetPlayerUserInfo(playerIndex byte, userInfo string) {
	st.SetPlayerName(playerIndex, infostring.Parse(userInfo).Get("name"))
}

func (st *Packet) ResetPlayerNames() {
	clear(st.playerNames[:])
}

func (st *Packet) CommitRawEntitySnapshot(sequenceByte byte, entities map[uint16]EntityRecord) {
	st.rawEntityHistory[sequenceByte] = CloneEntityMap(entities)
}

func (st *Packet) RawEntitySnapshot(sequenceByte byte) (map[uint16]EntityRecord, bool) {
	entities := st.rawEntityHistory[sequenceByte]
	if entities == nil {
		return nil, false
	}
	return entities, true
}

func (st *Packet) CommitEntitySnapshot(sequence uint32, entities map[uint16]EntityRecord) {
	history := &st.entityHistory[sequence&packetHistoryMask]
	history.sequence = sequence
	history.valid = true
	history.entities = CloneEntityMap(entities)
	history.ordered = sortedEntityRecords(history.entities)
}

func (st *Packet) CommitPacketEntities(records []EntityRecord, preserveRaw bool) {
	entities := make(map[uint16]EntityRecord, len(records))
	for _, record := range records {
		entities[EntityNumber(record)] = record
	}

	if preserveRaw {
		st.CommitRawEntitySnapshot(byte(st.packetSequence), entities)
	}
	st.CommitEntitySnapshot(st.packetSequence, entities)
	st.packetEntitiesCommitted = true
}

func (st *Packet) ResetEntityTracking() {
	clear(st.Baselines)

	for i := range st.entityHistory {
		st.entityHistory[i] = entityHistory{}
	}

	for i := range st.rawEntityHistory {
		st.rawEntityHistory[i] = nil
	}

	st.packetBaseSequence = 0
	st.hasPacketBase = false
	st.packetEntitiesCommitted = false
}

func (st *Packet) DefaultPlayer() PlayerRecord {
	var r PlayerRecord
	SetPlayerIndex(&r, st.PlayerIndex)
	SetPlayerModel(&r, st.PlayerModelIndex)
	return r
}
