package codec

// Effect bases use shared coordinate history or an entity's current origin.
// For damage, the entity is the command's target player.
const (
	effectBasePrevious byte = 0
	effectBaseEntity   byte = 1
)

// Both sides start each chunk with zeroed history so chunks decode independently.
type rangeCodecState struct {
	players                   [rangePlayerSlots]rangePlayerState
	entities                  [rangeEntitySlots]rangeEntityState
	previousPlayerIndex       byte
	previousPlayerIndexDelta  byte
	entityDeltaBase           byte
	previousRecordKind        byte
	haveRecordKind            bool
	previousTimestamp         byte
	previousPlayerRunLength   byte
	previousEntityCount       uint16
	previousSoundChannel      uint16
	previousSoundNumbers      [512]byte
	previousEffectCoordinates [3]uint32
	previousEffectEntity      uint16
	currentCommandTarget      byte
	previousStatIndex         [rangePlayerSlots]byte
	previousStatValues        [rangePlayerSlots][256]byte
	previousStatLongIndex     [rangePlayerSlots]byte
	previousStatLongValues    [rangePlayerSlots][256]uint32
	previousFixedPlayer       byte
	previousFrags             [rangePlayerSlots]uint16
	previousPings             [rangePlayerSlots]uint16
	previousPacketLoss        [rangePlayerSlots]byte
	currentTime               uint32
}

// Code the record kind and timestamp using the previous record's history, then
// advance the clock before coding the body. Motion predictors in that body use
// the new time. Raw bodies advance this clock too, but do not update field history.
func (state *rangeCodecState) updateRecordHistory(kind, timestamp byte) {
	state.currentTime += uint32(timestamp)
	state.previousRecordKind = kind
	state.haveRecordKind = true
	state.previousTimestamp = timestamp
}

// Classes select frequency rows. They are not stored as separate wire fields.
const (
	effectClassZero = iota
	effectClassPlayer
	effectClassOther
)

func effectEntityClass(entity uint16) int {
	switch {
	case entity == 0:
		return effectClassZero
	case entity <= rangePlayerSlots:
		return effectClassPlayer
	default:
		return effectClassOther
	}
}

// Player entity IDs start at one. All other IDs use the entity history.
func effectEntityCoords(state *rangeCodecState, entity uint16) ([3]uint32, bool) {
	if entity >= 1 && entity <= rangePlayerSlots {
		return state.players[entity-1].origin, true
	}
	if int(entity) < len(state.entities) {
		return state.entities[entity].origin, true
	}
	return [3]uint32{}, false
}
