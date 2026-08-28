package message

import "github.com/osm/quake/qizmo/state"

func playerCoordinates(
	players []state.PlayerRecord,
	entity uint16,
) ([3]uint16, bool) {
	for i := range players {
		if state.PlayerIndex(players[i]) == byte(entity-1) {
			return [3]uint16{
				uint16(players[i][1]),
				uint16(players[i][1] >> 16),
				uint16(players[i][2]),
			}, true
		}
	}

	return [3]uint16{}, false
}

func entityCoordinates(
	entities []state.EntityRecord,
	entity uint16,
) (coordinates [3]uint16, found, stoppedBefore bool) {
	for _, record := range entities {
		number := state.EntityNumber(record)
		if number == entity {
			return state.EntityOrigin(record), true, false
		}
		if entity < number {
			return [3]uint16{}, false, true
		}
	}

	return [3]uint16{}, false, false
}

func soundEntityCoordinates(
	st *state.Packet,
	entities []state.EntityRecord,
	entity uint16,
) ([3]uint16, bool) {
	if len(entities) == 0 {
		return [3]uint16{}, false
	}
	coordinates, found, stoppedBefore := entityCoordinates(entities, entity)
	if found {
		return coordinates, true
	}
	if !stoppedBefore {
		return [3]uint16{}, false
	}

	baseline, ok := st.Baselines[entity]
	if !ok {
		return [3]uint16{}, false
	}
	coordinates = state.EntityOrigin(baseline)
	return coordinates, coordinates != [3]uint16{}
}

func signedCoordinates(coordinates [3]uint16) [3]int16 {
	return [3]int16{
		int16(coordinates[0]),
		int16(coordinates[1]),
		int16(coordinates[2]),
	}
}

func unsignedCoordinates(coordinates [3]int16) [3]uint16 {
	return [3]uint16{
		uint16(coordinates[0]),
		uint16(coordinates[1]),
		uint16(coordinates[2]),
	}
}

// svc_nails packs 12-bit xyz, 4-bit pitch, and 8-bit yaw. Qizmo seeds its
// bytewise delta with the primary player's position and zero angles.
func nailProjectileBase(origin [3]uint16) [nailProjectileSize]byte {
	x, y, z := origin[0], origin[1], origin[2]
	return [nailProjectileSize]byte{
		byte(x >> 4),
		(byte(y) & 0xf0) | (byte(x>>8) >> 4),
		byte(y >> 8),
		byte(z >> 4),
		byte(int8(byte(z>>8)) >> 4),
		0,
	}
}
