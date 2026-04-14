package bsp

import (
	"strconv"
	"strings"
)

func (b *BSP) parseEntities(data []byte) error {
	entityData, err := lumpBytes(data, lumpEntities)
	if err != nil {
		return err
	}
	if len(entityData) == 0 {
		return nil
	}

	blocks := strings.Split(string(entityData), "}")
	b.Entities = make([]Entity, 0, len(blocks))
	for _, block := range blocks {
		block = strings.Trim(block, "\x00\r\n\t ")
		if block == "" {
			continue
		}
		block = strings.TrimPrefix(block, "{")
		entity := Entity{
			Index: len(b.Entities),
			Pairs: parseEntityFields(block),
		}
		entity.refreshDerived()
		b.Entities = append(b.Entities, entity)
	}
	return nil
}

func parseEntityFields(block string) []EntityField {
	var fields []EntityField
	var parts []string
	inQuote := false
	start := 0
	for i, r := range block {
		if r != '"' {
			continue
		}
		if !inQuote {
			inQuote = true
			start = i + 1
			continue
		}
		parts = append(parts, block[start:i])
		inQuote = false
	}
	for i := 0; i+1 < len(parts); i += 2 {
		fields = append(fields, EntityField{
			Key:   parts[i],
			Value: parts[i+1],
		})
	}
	return fields
}

func parseEntityOrigin(input string) [3]float64 {
	var origin [3]float64
	parts := strings.Fields(input)
	if len(parts) != 3 {
		return origin
	}
	for i := 0; i < 3; i++ {
		value, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			return origin
		}
		origin[i] = value
	}
	return origin
}
