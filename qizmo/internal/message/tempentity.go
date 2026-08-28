package message

import "github.com/osm/quake/protocol"

type tempEntityShape byte

const (
	tempEntityPoint tempEntityShape = iota
	tempEntityCountedPoint
	tempEntityBeam
)

const (
	tempEntityPointPayloadSize        = coordinateTripletSize
	tempEntityCountedPointPayloadSize = 1 + tempEntityPointPayloadSize
	tempEntityBeamPayloadSize         = 2 + 2*tempEntityPointPayloadSize
)

func tempEntityShapeForType(entityType byte) tempEntityShape {
	switch entityType {
	case protocol.TEGunshot, protocol.TEBlood:
		return tempEntityCountedPoint
	case protocol.TELightning1, protocol.TELightning2, protocol.TELightning3:
		return tempEntityBeam
	default:
		return tempEntityPoint
	}
}

func tempEntityPayloadSize(shape tempEntityShape) int {
	switch shape {
	case tempEntityCountedPoint:
		return tempEntityCountedPointPayloadSize
	case tempEntityBeam:
		return tempEntityBeamPayloadSize
	default:
		return tempEntityPointPayloadSize
	}
}
