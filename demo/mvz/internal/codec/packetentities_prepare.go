package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/deltapacketentities"
	"github.com/osm/quake/packet/command/packetentities"
	"github.com/osm/quake/packet/command/packetentity"
	"github.com/osm/quake/packet/command/packetentitydelta"
	"github.com/osm/quake/protocol"
	"github.com/osm/quake/protocol/fte"
	mvdprotocol "github.com/osm/quake/protocol/mvd"
)

func validatePacketEntityCount(parsed command.Command) error {
	var count int
	switch typed := parsed.(type) {
	case *packetentities.Command:
		count = len(typed.Entities)
	case *deltapacketentities.Command:
		count = len(typed.Entities)
	default:
		return nil
	}
	if count <= maxPacketEntityCount {
		return nil
	}
	return fmt.Errorf(
		"packet-entity count %d exceeds MVZ limit %d",
		count,
		maxPacketEntityCount,
	)
}

// Only use structured encoding when parsed fields reproduce the original bytes.
// Unfamiliar extension layouts must survive unchanged in the general stream.
func newPacketEntities(
	raw []byte,
	full *packetentities.Command,
	delta *deltapacketentities.Command,
) *rangePacketEntities {
	var (
		base     byte
		commands []*packetentity.Command
		isDelta  bool
		offset   int
	)
	switch {
	case full != nil && delta == nil && len(raw) >= 1 &&
		raw[0] == protocol.SVCPacketEntities:
		commands = full.Entities
		offset = 1
	case full == nil && delta != nil && len(raw) >= 2 &&
		raw[0] == protocol.SVCDeltaPacketEntities && raw[1] == delta.Index:
		base = delta.Index
		commands = delta.Entities
		isDelta = true
		offset = 2
	default:
		return nil
	}
	if len(commands) > maxPacketEntityCount {
		return nil
	}

	packet := &rangePacketEntities{
		delta:    isDelta,
		base:     base,
		entities: make([]rangePacketEntity, 0, len(commands)),
	}
	for _, command := range commands {
		entity, ok := newPacketEntity(raw[offset:], command)
		if !ok {
			return nil
		}
		offset += len(entity.raw)
		packet.entities = append(packet.entities, entity)
	}
	if len(raw)-offset != 2 || binary.LittleEndian.Uint16(raw[offset:]) != 0 {
		return nil
	}
	return packet
}

// raw starts at this entity. Later entities and the terminator may follow.
func newPacketEntity(
	raw []byte,
	command *packetentity.Command,
) (rangePacketEntity, bool) {
	if command == nil {
		return rangePacketEntity{}, false
	}
	wire := command.Bytes()
	if len(wire) < 2 || len(wire) > len(raw) ||
		!bytes.Equal(wire, raw[:len(wire)]) ||
		binary.LittleEndian.Uint16(wire) != command.Bits ||
		command.Bits == 0 {
		return rangePacketEntity{}, false
	}
	entity := rangePacketEntity{raw: raw[:len(wire)]}
	if !setEntityLayout(&entity, command) {
		return rangePacketEntity{}, false
	}
	if _, err := entity.layout(); err != nil {
		return rangePacketEntity{}, false
	}
	return entity, true
}

func setEntityLayout(
	entity *rangePacketEntity,
	command *packetentity.Command,
) bool {
	if command.Bits&protocol.URemove != 0 {
		entity.flagCount = len(entity.raw) - 2
		return true
	}
	delta := command.PacketEntityDelta
	if delta == nil {
		return false
	}
	entity.coordSize = int(delta.CoordSize)
	if delta.MVDProtocolExtension&mvdprotocol.ExtensionFloatCoords != 0 {
		entity.coordSize = 4
	}
	entity.angleSize = int(delta.AngleSize)
	entity.flagCount = entityFlagByteCount(command.Bits, delta)
	return true
}

func entityFlagByteCount(
	bits uint16,
	delta *packetentitydelta.Command,
) int {
	if bits&protocol.UMoreBits == 0 {
		return 0
	}
	count := 1
	if delta.FTEProtocolExtension == 0 || delta.MoreBits&fte.UEvenMore == 0 {
		return count
	}
	count++
	if delta.EvenMoreBits&fte.UYetMore != 0 {
		count++
	}
	return count
}
