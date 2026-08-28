package message

import (
	"testing"

	"github.com/osm/quake/protocol"
)

func TestPacketEntityFieldMetadataCoversEachByte(t *testing.T) {
	var masks byte
	for i, field := range packetEntityFields {
		if field.offset != 4+i {
			t.Fatalf("field %d offset = %d, want %d", i, field.offset, 4+i)
		}
		if masks&field.mask != 0 {
			t.Fatalf("field %d repeats mask %#x", i, field.mask)
		}
		masks |= field.mask
	}
	if masks != 0xff {
		t.Fatalf("combined packet-entity field mask = %#x, want 0xff", masks)
	}
}

func TestTempEntityShapes(t *testing.T) {
	tests := []struct {
		entityType byte
		shape      tempEntityShape
		payload    int
	}{
		{protocol.TEGunshot, tempEntityCountedPoint, 7},
		{protocol.TEBlood, tempEntityCountedPoint, 7},
		{protocol.TELightning1, tempEntityBeam, 14},
		{protocol.TELightning2, tempEntityBeam, 14},
		{protocol.TELightning3, tempEntityBeam, 14},
		{protocol.TEExplosion, tempEntityPoint, 6},
	}
	for _, test := range tests {
		shape := tempEntityShapeForType(test.entityType)
		if shape != test.shape {
			t.Fatalf("type %d shape = %d, want %d", test.entityType, shape, test.shape)
		}
		if payload := tempEntityPayloadSize(shape); payload != test.payload {
			t.Fatalf("type %d payload = %d, want %d", test.entityType, payload, test.payload)
		}
	}
}
