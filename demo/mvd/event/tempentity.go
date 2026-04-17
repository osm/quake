package event

import "github.com/osm/quake/packet/command/tempentity"

func (p *parser) handleTempEntity(cmd *tempentity.Command) {
	if cmd == nil {
		return
	}

	p.appendTempEntity(TempEntity{
		Time: p.elapsed,
		Kind: cmd.Type,
		Pos: Vec3{
			X: cmd.Coord[0],
			Y: cmd.Coord[1],
			Z: cmd.Coord[2],
		},
		EndPos: Vec3{
			X: cmd.EndCoord[0],
			Y: cmd.EndCoord[1],
			Z: cmd.EndCoord[2],
		},
		Entity:   cmd.Entity,
		Count:    cmd.Count,
		ColorA:   cmd.ColorStart,
		ColorLen: cmd.ColorLength,
	})
}
