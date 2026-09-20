package codec

import (
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/protocol"
)

type rangePacketEntity struct {
	raw       []byte
	flagCount int
	coordSize int
	angleSize int
}

type rangeEntityLayout struct {
	header  uint16
	flags   []byte
	bits    uint16
	payload []byte
	tail    []byte
}

func (entity rangePacketEntity) layout() (rangeEntityLayout, error) {
	if err := entity.validateEnvelope(); err != nil {
		return rangeEntityLayout{}, err
	}
	header := binary.LittleEndian.Uint16(entity.raw)
	if header == 0 {
		return rangeEntityLayout{}, fmt.Errorf("packet-entity uses terminator header")
	}
	flags := entity.raw[2 : 2+entity.flagCount]
	bits := header &^ protocol.UCheckMoreBits
	if len(flags) != 0 {
		bits |= uint16(flags[0])
	}
	if header&protocol.URemove != 0 {
		if len(entity.raw) != 2+entity.flagCount {
			return rangeEntityLayout{}, fmt.Errorf("removed packet-entity has payload")
		}
		return rangeEntityLayout{header: header, flags: flags, bits: bits}, nil
	}
	if err := entity.validateFieldWidths(); err != nil {
		return rangeEntityLayout{}, err
	}

	offset := entity.fieldEnd(bits)
	if offset > len(entity.raw) {
		return rangeEntityLayout{}, fmt.Errorf("packet-entity fields exceed wire data")
	}
	return rangeEntityLayout{
		header:  header,
		flags:   flags,
		bits:    bits,
		payload: entity.raw[2+entity.flagCount : offset],
		tail:    entity.raw[offset:],
	}, nil
}

func (entity rangePacketEntity) validateEnvelope() error {
	if len(entity.raw) < 2 || entity.flagCount < 0 || entity.flagCount > 3 ||
		2+entity.flagCount > len(entity.raw) {
		return fmt.Errorf("invalid packet-entity envelope")
	}
	return nil
}

func (entity rangePacketEntity) validateFieldWidths() error {
	if !validCoordSize(entity.coordSize) {
		return fmt.Errorf(
			"invalid packet-entity coordinate size %d",
			entity.coordSize,
		)
	}
	if !validEntityAngleSize(entity.angleSize) {
		return fmt.Errorf(
			"invalid packet-entity angle size %d",
			entity.angleSize,
		)
	}
	return nil
}

func (entity rangePacketEntity) fieldEnd(bits uint16) int {
	offset := 2 + entity.flagCount
	for _, mask := range [...]uint16{
		protocol.UModel,
		protocol.UFrame,
		protocol.UColorMap,
		protocol.USkin,
		protocol.UEffects,
	} {
		if bits&mask != 0 {
			offset++
		}
	}
	for axis := 0; axis < 3; axis++ {
		if bits&(protocol.UOrigin1<<axis) != 0 {
			offset += entity.coordSize
		}
		if bits&entityAngleMask(axis) != 0 {
			offset += entity.angleSize
		}
	}
	return offset
}

func validEntityAngleSize(size int) bool {
	return size == 1 || size == 2
}

func entityAngleMask(axis int) uint16 {
	switch axis {
	case 0:
		return protocol.UAngle1
	case 1:
		return protocol.UAngle2
	default:
		return protocol.UAngle3
	}
}
