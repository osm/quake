package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/osm/quake/demo/mvd"
	"github.com/osm/quake/packet/command"
	"github.com/osm/quake/packet/command/centerprint"
	"github.com/osm/quake/packet/command/damage"
	"github.com/osm/quake/packet/command/deltapacketentities"
	"github.com/osm/quake/packet/command/packetentities"
	"github.com/osm/quake/packet/command/playerinfo"
	"github.com/osm/quake/packet/command/print"
	"github.com/osm/quake/packet/command/sound"
	"github.com/osm/quake/packet/command/stufftext"
	"github.com/osm/quake/packet/command/tempentity"
	"github.com/osm/quake/packet/svc"
	mvdprotocol "github.com/osm/quake/protocol/mvd"
)

// rangeRecord keeps the original payload even when its body can be expressed
// as structured operations. Raw records preserve that payload verbatim.
// Structured records preserve the prefix and any unmodeled command bytes.
type rangeRecord struct {
	timestamp  byte
	command    byte
	payload    []byte
	structured bool
	prefix     []byte
	operations []rangeOperation
}

type rangeOperation struct {
	raw    []byte
	player *rangePlayerInfo
	packet *rangePacketEntities
	sound  *rangeSound
	temp   *rangeTempEntity
	fixed  *rangeFixedOperation
	text   *rangeTextOperation
	damage *rangeDamage
}

type rangeChunkInput struct {
	records  []rangeRecord
	trailing []byte
	original []byte
}

func (chunk parsedRangeChunk) prepare() (rangeChunkInput, error) {
	prepared := rangeChunkInput{
		records:  make([]rangeRecord, len(chunk.records)),
		trailing: chunk.trailing,
		original: chunk.original,
	}
	for i := range chunk.records {
		record, err := newRecord(&chunk.records[i])
		if err != nil {
			return rangeChunkInput{}, fmt.Errorf(
				"prepare MVD record %d: %w", chunk.firstRecord+i, err,
			)
		}
		prepared.records[i] = record
	}
	return prepared, nil
}

func newRecord(data *mvd.Data) (rangeRecord, error) {
	raw := data.OriginalBytes()
	if raw == nil {
		raw = data.Bytes()
	}
	if len(raw) < 2 {
		return rangeRecord{}, fmt.Errorf("record is shorter than its header")
	}
	record := rangeRecord{timestamp: raw[0], command: raw[1], payload: raw[2:]}
	if data.Read == nil {
		return record, nil
	}
	gameData, ok := data.Read.Packet.(*svc.GameData)
	if !ok || len(gameData.Commands) != len(gameData.RawCmds) {
		return record, nil
	}
	body, prefixLength, valid, err := recordBody(record, data, gameData)
	if err != nil {
		return rangeRecord{}, err
	}
	if !valid {
		return record, nil
	}
	operations, hasStructured, err := prepareOperations(body, gameData)
	if err != nil {
		return rangeRecord{}, err
	}
	if !hasStructured {
		return record, nil
	}
	record.structured = true
	record.prefix = record.payload[:prefixLength]
	record.operations = operations
	return record, nil
}

func recordBody(
	record rangeRecord,
	data *mvd.Data,
	gameData *svc.GameData,
) ([]byte, int, bool, error) {
	prefixLength := 0
	if data.Command&0x7 == mvdprotocol.DemoMultiple {
		prefixLength = 4
	}
	if len(record.payload) < prefixLength+4 {
		return nil, 0, false, fmt.Errorf("read payload is shorter than its envelope")
	}
	body := record.payload[prefixLength+4:]
	wireSize := binary.LittleEndian.Uint32(record.payload[prefixLength : prefixLength+4])
	if uint64(wireSize) != uint64(len(body)) || data.Read.Size != wireSize ||
		!commandsMatchBody(body, gameData.RawCmds) {
		return nil, 0, false, nil
	}
	return body, prefixLength, true, nil
}

func prepareOperations(
	body []byte,
	gameData *svc.GameData,
) ([]rangeOperation, bool, error) {
	var operations []rangeOperation
	rawStart := -1
	commandOffset := 0
	hasStructured := false
	for i, parsedCommand := range gameData.Commands {
		if err := validatePacketEntityCount(parsedCommand); err != nil {
			return nil, false, err
		}
		rawCommand := gameData.RawCmds[i]
		operation := newOperation(parsedCommand, rawCommand)
		if !operation.structured() {
			if rawStart < 0 {
				rawStart = commandOffset
			}
			commandOffset += len(rawCommand)
			continue
		}
		if operations == nil {
			// Reserve once for ordinary packets. Long raw command runs
			// coalesce, so do not reserve an operation for every raw byte.
			operations = make([]rangeOperation, 0, min(len(gameData.Commands), 64))
		}
		operations = appendRawOperation(
			operations, body, rawStart, commandOffset,
		)
		rawStart = -1
		operations = append(operations, operation)
		commandOffset += len(rawCommand)
		hasStructured = true
	}
	operations = appendRawOperation(
		operations, body, rawStart, commandOffset,
	)
	return operations, hasStructured, nil
}

func appendRawOperation(
	operations []rangeOperation,
	body []byte,
	start, end int,
) []rangeOperation {
	if start < 0 || start >= end {
		return operations
	}
	operation := rangeOperation{raw: body[start:end]}
	return append(operations, operation)
}

func commandsMatchBody(body []byte, commands [][]byte) bool {
	offset := 0
	for _, command := range commands {
		if len(command) > len(body)-offset ||
			!bytes.Equal(command, body[offset:offset+len(command)]) {
			return false
		}
		offset += len(command)
	}
	return offset == len(body)
}

func newOperation(
	parsedCommand command.Command,
	rawCommand []byte,
) rangeOperation {
	var operation rangeOperation
	switch typed := parsedCommand.(type) {
	case *playerinfo.Command:
		if typed.IsMVD && typed.MVD != nil &&
			validPlayerInfo(rawCommand, int(typed.MVD.CoordSize)) {
			operation.player = &rangePlayerInfo{
				raw:       rawCommand,
				coordSize: int(typed.MVD.CoordSize),
			}
		}
	case *packetentities.Command:
		operation.packet = newPacketEntities(rawCommand, typed, nil)
	case *deltapacketentities.Command:
		operation.packet = newPacketEntities(rawCommand, nil, typed)
	case *sound.Command:
		operation.sound = newSound(rawCommand, typed)
	case *tempentity.Command:
		operation.temp = newTempEntity(rawCommand, typed)
	case *centerprint.Command, *stufftext.Command, *print.Command:
		operation.text = newTextOperation(rawCommand)
	case *damage.Command:
		operation.damage = newDamage(rawCommand, typed)
	}
	if !operation.structured() {
		operation.fixed = newFixedOperation(rawCommand)
	}
	return operation
}

func (operation rangeOperation) structured() bool {
	return operation.player != nil || operation.packet != nil || operation.sound != nil ||
		operation.temp != nil || operation.fixed != nil || operation.text != nil ||
		operation.damage != nil
}
