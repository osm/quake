package bsp

import (
	"encoding/binary"
	"math"
	"sort"
	"strings"
)

func (b *BSP) Bytes() []byte {
	var lumps [headerLumpCount][]byte

	lumps[lumpEntities] = b.entityBytes()
	lumps[lumpPlanes] = b.planeBytes()
	lumps[lumpMiptex] = append([]byte(nil), b.Miptex...)
	lumps[lumpVertices] = b.vertexBytes()
	lumps[lumpVisibility] = append([]byte(nil), b.Visibility...)
	lumps[lumpNodes] = b.nodeBytes()
	lumps[lumpTexinfo] = b.texinfoBytes()
	lumps[lumpFaces] = b.faceBytes()
	lumps[lumpLighting] = append([]byte(nil), b.Lighting...)
	lumps[lumpClipnodes] = b.clipnodeBytes()
	lumps[lumpLeafs] = b.leafBytes()
	lumps[lumpMarksurfaces] = b.marksurfaceBytes()
	lumps[lumpEdges] = b.edgeBytes()
	lumps[lumpSurfedges] = b.surfedgeBytes()
	lumps[lumpModels] = b.modelBytes()

	size := 4 + headerLumpCount*8

	for _, lump := range lumps {
		size = align4(size)
		size += len(lump)
	}

	data := make([]byte, size)
	binary.LittleEndian.PutUint32(data[0:4], uint32(b.Version))

	order := b.lumpOrder()
	offset := align4(4 + headerLumpCount*8)
	offsets := [headerLumpCount]int{}

	for _, lumpIndex := range order {
		offset = align4(offset)
		offsets[lumpIndex] = offset
		copy(data[offset:], lumps[lumpIndex])
		offset += len(lumps[lumpIndex])
	}

	for i, lump := range lumps {
		headerOffset := 4 + i*8
		binary.LittleEndian.PutUint32(data[headerOffset:], uint32(offsets[i]))
		binary.LittleEndian.PutUint32(data[headerOffset+4:], uint32(len(lump)))
	}

	return data
}

func align4(n int) int {
	remainder := n % 4
	if remainder == 0 {
		return n
	}
	return n + (4 - remainder)
}

func (b *BSP) entityBytes() []byte {
	var out strings.Builder
	for _, entity := range b.Entities {
		out.WriteString("{\n")
		for _, field := range entity.Pairs {
			out.WriteByte('"')
			out.WriteString(field.Key)
			out.WriteString(`" "`)
			out.WriteString(field.Value)
			out.WriteString("\"\n")
		}
		out.WriteString("}\n")
	}
	out.WriteByte(0)
	return []byte(out.String())
}

func (b *BSP) lumpOrder() []int {
	type lumpOrderEntry struct {
		index  int
		offset int
	}

	entries := make([]lumpOrderEntry, 0, headerLumpCount)
	for i := 0; i < headerLumpCount; i++ {
		entries = append(entries, lumpOrderEntry{
			index:  i,
			offset: b.offsets[i],
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]

		if left.offset == 0 && right.offset == 0 {
			return left.index < right.index
		}

		if left.offset == 0 {
			return false
		}

		if right.offset == 0 {
			return true
		}

		return left.offset < right.offset
	})

	order := make([]int, 0, len(entries))
	for _, entry := range entries {
		order = append(order, entry.index)
	}

	return order
}

func (b *BSP) planeBytes() []byte {
	data := make([]byte, len(b.Planes)*planeSize)
	for planeIndex, plane := range b.Planes {
		offset := planeIndex * planeSize
		putFloat32(data[offset:], plane.Normal[0])
		putFloat32(data[offset+4:], plane.Normal[1])
		putFloat32(data[offset+8:], plane.Normal[2])
		putFloat32(data[offset+12:], plane.Dist)
		binary.LittleEndian.PutUint32(data[offset+16:], uint32(plane.Type))
	}
	return data
}

func (b *BSP) vertexBytes() []byte {
	data := make([]byte, len(b.Vertices)*vertexSize)
	for i, vertex := range b.Vertices {
		offset := i * vertexSize
		putFloat32(data[offset:], vertex.X)
		putFloat32(data[offset+4:], vertex.Y)
		putFloat32(data[offset+8:], vertex.Z)
	}
	return data
}

func (b *BSP) nodeBytes() []byte {
	data := make([]byte, len(b.Nodes)*nodeSize)
	for nodeIndex, node := range b.Nodes {
		offset := nodeIndex * nodeSize
		binary.LittleEndian.PutUint32(data[offset:], uint32(node.Plane))
		binary.LittleEndian.PutUint16(data[offset+4:], uint16(node.Children[0]))
		binary.LittleEndian.PutUint16(data[offset+6:], uint16(node.Children[1]))
		putInt16s(data[offset+8:], node.Mins[:])
		putInt16s(data[offset+14:], node.Maxs[:])
		binary.LittleEndian.PutUint16(data[offset+20:], node.FirstFace)
		binary.LittleEndian.PutUint16(data[offset+22:], node.NumFaces)
	}
	return data
}

func (b *BSP) texinfoBytes() []byte {
	data := make([]byte, len(b.Texinfo)*texinfoSize)
	for texinfoIndex, texinfo := range b.Texinfo {
		offset := texinfoIndex * texinfoSize
		for row := 0; row < 2; row++ {
			for col := 0; col < 4; col++ {
				putFloat32(
					data[offset+row*16+col*4:],
					texinfo.Vecs[row][col],
				)
			}
		}
		binary.LittleEndian.PutUint32(data[offset+32:], uint32(texinfo.Miptex))
		binary.LittleEndian.PutUint32(data[offset+36:], uint32(texinfo.Flags))
	}
	return data
}

func (b *BSP) faceBytes() []byte {
	data := make([]byte, len(b.Faces)*faceSize)
	for faceIndex, face := range b.Faces {
		offset := faceIndex * faceSize
		binary.LittleEndian.PutUint16(data[offset:], face.Plane)
		binary.LittleEndian.PutUint16(data[offset+2:], face.Side)
		binary.LittleEndian.PutUint32(data[offset+4:], uint32(face.FirstEdge))
		binary.LittleEndian.PutUint16(data[offset+8:], face.NumEdges)
		binary.LittleEndian.PutUint16(data[offset+10:], face.Texinfo)
		copy(data[offset+12:offset+16], face.Styles[:])
		binary.LittleEndian.PutUint32(data[offset+16:], uint32(face.LightOfs))
	}
	return data
}

func (b *BSP) clipnodeBytes() []byte {
	data := make([]byte, len(b.Clipnodes)*clipnodeSize)
	for clipnodeIndex, clipnode := range b.Clipnodes {
		offset := clipnodeIndex * clipnodeSize
		binary.LittleEndian.PutUint32(data[offset:], uint32(clipnode.Plane))
		binary.LittleEndian.PutUint16(data[offset+4:], uint16(clipnode.Children[0]))
		binary.LittleEndian.PutUint16(data[offset+6:], uint16(clipnode.Children[1]))
	}
	return data
}

func (b *BSP) leafBytes() []byte {
	data := make([]byte, len(b.Leafs)*leafSize)
	for leafIndex, leaf := range b.Leafs {
		offset := leafIndex * leafSize
		binary.LittleEndian.PutUint32(data[offset:], uint32(leaf.Contents))
		binary.LittleEndian.PutUint32(
			data[offset+4:],
			uint32(leaf.VisibilityOffset),
		)
		putInt16s(data[offset+8:], leaf.Mins[:])
		putInt16s(data[offset+14:], leaf.Maxs[:])
		binary.LittleEndian.PutUint16(data[offset+20:], leaf.FirstMarkSurface)
		binary.LittleEndian.PutUint16(data[offset+22:], leaf.NumMarkSurfaces)
		copy(data[offset+24:offset+28], leaf.AmbientLevel[:])
	}
	return data
}

func (b *BSP) marksurfaceBytes() []byte {
	data := make([]byte, len(b.Marksurfaces)*marksurfaceSize)
	for i, marksurface := range b.Marksurfaces {
		binary.LittleEndian.PutUint16(data[i*marksurfaceSize:], marksurface)
	}
	return data
}

func (b *BSP) edgeBytes() []byte {
	data := make([]byte, len(b.Edges)*edgeSize)
	for i, edge := range b.Edges {
		offset := i * edgeSize
		binary.LittleEndian.PutUint16(data[offset:], edge.A)
		binary.LittleEndian.PutUint16(data[offset+2:], edge.B)
	}
	return data
}

func (b *BSP) surfedgeBytes() []byte {
	data := make([]byte, len(b.Surfedges)*4)
	for i, surfedge := range b.Surfedges {
		binary.LittleEndian.PutUint32(data[i*4:], uint32(surfedge))
	}
	return data
}

func (b *BSP) modelBytes() []byte {
	data := make([]byte, len(b.Models)*modelSize)
	for modelIndex, model := range b.Models {
		offset := modelIndex * modelSize
		putFloat32(data[offset:], model.Bounds.Min[0])
		putFloat32(data[offset+4:], model.Bounds.Min[1])
		putFloat32(data[offset+8:], model.Bounds.Min[2])
		putFloat32(data[offset+12:], model.Bounds.Max[0])
		putFloat32(data[offset+16:], model.Bounds.Max[1])
		putFloat32(data[offset+20:], model.Bounds.Max[2])
		putFloat32(data[offset+24:], model.Origin[0])
		putFloat32(data[offset+28:], model.Origin[1])
		putFloat32(data[offset+32:], model.Origin[2])
		for j := 0; j < 4; j++ {
			binary.LittleEndian.PutUint32(
				data[offset+36+j*4:],
				uint32(model.Headnodes[j]),
			)
		}
		binary.LittleEndian.PutUint32(data[offset+52:], uint32(model.Visleafs))
		binary.LittleEndian.PutUint32(data[offset+56:], uint32(model.FirstFace))
		binary.LittleEndian.PutUint32(data[offset+60:], uint32(model.NumFaces))
	}
	return data
}

func putInt16s(dst []byte, values []int16) {
	for i, value := range values {
		binary.LittleEndian.PutUint16(dst[i*2:], uint16(value))
	}
}

func putFloat32(dst []byte, value float32) {
	binary.LittleEndian.PutUint32(dst, math.Float32bits(value))
}
