package event

import (
	"strings"

	"github.com/osm/quake/common/item"
	"github.com/osm/quake/packet/command/deltapacketentities"
	"github.com/osm/quake/packet/command/modellist"
	"github.com/osm/quake/packet/command/packetentities"
	"github.com/osm/quake/packet/command/packetentity"
	"github.com/osm/quake/packet/command/spawnbaseline"
	"github.com/osm/quake/protocol"
)

type entityStateReconstructor struct {
	modelNames      []string
	baselines       map[uint16]entityState
	currentEntities map[uint16]entityState
}

type entityState struct {
	ModelIndex byte
	Pos        Vec3
	Angles     Vec3
}

type entityUpdate struct {
	Number uint16
	Remove bool

	HasModel bool
	Model    byte

	HasCoord [3]bool
	Coord    [3]float32

	HasAngle [3]bool
	Angle    [3]float32
}

func newEntityStateReconstructor() *entityStateReconstructor {
	return &entityStateReconstructor{
		modelNames:      []string{""},
		baselines:       make(map[uint16]entityState),
		currentEntities: make(map[uint16]entityState),
	}
}

func (r *entityStateReconstructor) recordModelList(cmd *modellist.Command) {
	if cmd == nil || len(cmd.Models) == 0 {
		return
	}
	r.modelNames = append(r.modelNames, cmd.Models...)
}

func (r *entityStateReconstructor) recordSpawnBaseline(cmd *spawnbaseline.Command) {
	if cmd == nil || cmd.Baseline == nil {
		return
	}

	r.baselines[cmd.Index] = entityState{
		ModelIndex: cmd.Baseline.ModelIndex,
		Pos: Vec3{
			X: cmd.Baseline.Coord[0],
			Y: cmd.Baseline.Coord[1],
			Z: cmd.Baseline.Coord[2],
		},
		Angles: Vec3{
			X: cmd.Baseline.Angle[0],
			Y: cmd.Baseline.Angle[1],
			Z: cmd.Baseline.Angle[2],
		},
	}
}

func (r *entityStateReconstructor) projectilesFromPacketEntities(
	cmd *packetentities.Command,
) []Projectile {
	if cmd == nil {
		return nil
	}

	updates := entityUpdatesFromPacketEntities(cmd)

	next := make(map[uint16]entityState)
	for _, update := range updates {
		if update.Remove {
			continue
		}
		base := r.baselines[update.Number]
		next[update.Number] = applyEntityUpdate(base, update)
	}

	r.currentEntities = next
	return r.projectiles()
}

func (r *entityStateReconstructor) projectilesFromDeltaPacketEntities(
	cmd *deltapacketentities.Command,
) []Projectile {
	if cmd == nil {
		return nil
	}

	updates := entityUpdatesFromDeltaPacketEntities(cmd)

	next := make(map[uint16]entityState, len(r.currentEntities))
	for entity, state := range r.currentEntities {
		next[entity] = state
	}

	for _, update := range updates {
		if update.Remove {
			delete(next, update.Number)
			continue
		}

		base, ok := next[update.Number]
		if !ok {
			base = r.baselines[update.Number]
		}
		next[update.Number] = applyEntityUpdate(base, update)
	}

	r.currentEntities = next
	return r.projectiles()
}

func (r *entityStateReconstructor) projectiles() []Projectile {
	out := make([]Projectile, 0)
	for entityNumber, state := range r.currentEntities {
		modelName := r.modelName(int(state.ModelIndex))
		weapon, ok := projectileWeaponFromModelName(modelName)
		if !ok {
			continue
		}

		out = append(out, Projectile{
			Entity: int(entityNumber),
			Pos:    state.Pos,
			Angles: state.Angles,
			Model:  modelName,
			Weapon: weapon,
		})
	}
	return out
}

func (r *entityStateReconstructor) modelName(index int) string {
	if index < 0 || index >= len(r.modelNames) {
		return ""
	}
	return strings.TrimSpace(r.modelNames[index])
}

func entityUpdatesFromPacketEntities(
	cmd *packetentities.Command,
) []entityUpdate {
	if cmd == nil || len(cmd.Entities) == 0 {
		return nil
	}

	updates := make([]entityUpdate, 0, len(cmd.Entities))
	for _, entity := range cmd.Entities {
		updates = append(updates, entityUpdateFromPacketEntity(entity))
	}
	return updates
}

func entityUpdatesFromDeltaPacketEntities(
	cmd *deltapacketentities.Command,
) []entityUpdate {
	if cmd == nil || len(cmd.Entities) == 0 {
		return nil
	}

	updates := make([]entityUpdate, 0, len(cmd.Entities))
	for _, entity := range cmd.Entities {
		updates = append(updates, entityUpdateFromPacketEntity(entity))
	}
	return updates
}

func entityUpdateFromPacketEntity(cmd *packetentity.Command) entityUpdate {
	if cmd == nil {
		return entityUpdate{}
	}

	update := entityUpdate{
		Number: cmd.Bits & 511,
	}

	bits := cmd.Bits &^ 511
	if bits&protocol.URemove != 0 {
		update.Remove = true
		return update
	}

	delta := cmd.PacketEntityDelta
	if delta == nil {
		return update
	}

	if bits&protocol.UMoreBits != 0 {
		bits |= uint16(delta.MoreBits)
	}

	if bits&protocol.UModel != 0 {
		update.HasModel = true
		update.Model = delta.ModelIndex
	}

	for axis, bit := range [3]uint16{
		protocol.UOrigin1,
		protocol.UOrigin2,
		protocol.UOrigin3,
	} {
		if bits&bit == 0 {
			continue
		}
		update.HasCoord[axis] = true
		update.Coord[axis] = delta.Coord[axis]
	}

	for axis, bit := range [3]uint16{
		protocol.UAngle1,
		protocol.UAngle2,
		protocol.UAngle3,
	} {
		if bits&bit == 0 {
			continue
		}
		update.HasAngle[axis] = true
		update.Angle[axis] = delta.Angle[axis]
	}

	return update
}

func applyEntityUpdate(base entityState, update entityUpdate) entityState {
	state := base
	if update.HasModel {
		state.ModelIndex = update.Model
	}

	for axis := 0; axis < 3; axis++ {
		if update.HasCoord[axis] {
			switch axis {
			case 0:
				state.Pos.X = update.Coord[0]
			case 1:
				state.Pos.Y = update.Coord[1]
			case 2:
				state.Pos.Z = update.Coord[2]
			}
		}
		if update.HasAngle[axis] {
			switch axis {
			case 0:
				state.Angles.X = update.Angle[0]
			case 1:
				state.Angles.Y = update.Angle[1]
			case 2:
				state.Angles.Z = update.Angle[2]
			}
		}
	}

	return state
}

func projectileWeaponFromModelName(name string) (item.Item, bool) {
	lowered := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.HasSuffix(lowered, "missile.mdl"):
		return item.Item{Short: "RL", Long: "Rocket Launcher"}, true
	case strings.HasSuffix(lowered, "grenade.mdl"):
		return item.Item{Short: "GL", Long: "Grenade Launcher"}, true
	default:
		return item.Item{}, false
	}
}
