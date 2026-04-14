package bsp

import (
	"encoding/binary"
	"fmt"
	"math"
)

func Parse(data []byte) (*BSP, error) {
	if len(data) < 4+headerLumpCount*8 {
		return nil, fmt.Errorf("bsp file too small")
	}

	version := int32(binary.LittleEndian.Uint32(data[0:4]))
	if version != versionQuake {
		return nil, fmt.Errorf("unsupported BSP version: %d", version)
	}

	b := &BSP{Version: version}

	for i := 0; i < headerLumpCount; i++ {
		offset, _ := lumpRange(data, i)
		b.offsets[i] = offset
	}

	// Parse each lump into the typed BSP structure.
	if err := b.parseEntities(data); err != nil {
		return nil, err
	}
	if err := b.parsePlanes(data); err != nil {
		return nil, err
	}
	if err := b.parseMiptex(data); err != nil {
		return nil, err
	}
	if err := b.parseVertices(data); err != nil {
		return nil, err
	}
	if err := b.parseVisibility(data); err != nil {
		return nil, err
	}
	if err := b.parseNodes(data); err != nil {
		return nil, err
	}
	if err := b.parseTexinfo(data); err != nil {
		return nil, err
	}
	if err := b.parseFaces(data); err != nil {
		return nil, err
	}
	if err := b.parseLighting(data); err != nil {
		return nil, err
	}
	if err := b.parseClipnodes(data); err != nil {
		return nil, err
	}
	if err := b.parseLeafs(data); err != nil {
		return nil, err
	}
	if err := b.parseMarksurfaces(data); err != nil {
		return nil, err
	}
	if err := b.parseEdges(data); err != nil {
		return nil, err
	}
	if err := b.parseSurfedges(data); err != nil {
		return nil, err
	}
	if err := b.parseModels(data); err != nil {
		return nil, err
	}

	return b, nil
}

func lumpRange(data []byte, index int) (int, int) {
	offset := 4 + index*8
	return int(binary.LittleEndian.Uint32(data[offset:])),
		int(binary.LittleEndian.Uint32(data[offset+4:]))
}

func lumpBytes(data []byte, index int) ([]byte, error) {
	offset, length := lumpRange(data, index)
	if length < 0 || offset < 0 || offset+length > len(data) {
		return nil, fmt.Errorf("invalid BSP lump %d", index)
	}
	return append([]byte(nil), data[offset:offset+length]...), nil
}

func float32FromBytes(data []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(data))
}
