package bsp

import "encoding/binary"

func (b *BSP) parsePlanes(data []byte) error {
	planeData, err := lumpBytes(data, lumpPlanes)
	if err != nil {
		return err
	}

	b.Planes = make([]Plane, 0, len(planeData)/planeSize)
	for planeOffset := 0; planeOffset+planeSize <= len(planeData); planeOffset += planeSize {
		b.Planes = append(b.Planes, Plane{
			Normal: [3]float32{
				float32FromBytes(planeData[planeOffset:]),
				float32FromBytes(planeData[planeOffset+4:]),
				float32FromBytes(planeData[planeOffset+8:]),
			},
			Dist: float32FromBytes(planeData[planeOffset+12:]),
			Type: int32(binary.LittleEndian.Uint32(planeData[planeOffset+16:])),
		})
	}
	return nil
}

func (b *BSP) parseMiptex(data []byte) error {
	miptexData, err := lumpBytes(data, lumpMiptex)
	if err != nil {
		return err
	}
	b.Miptex = miptexData
	return nil
}

func (b *BSP) parseVertices(data []byte) error {
	vertexData, err := lumpBytes(data, lumpVertices)
	if err != nil {
		return err
	}
	b.Vertices = make([]Vertex, 0, len(vertexData)/vertexSize)
	for i := 0; i+vertexSize <= len(vertexData); i += vertexSize {
		b.Vertices = append(b.Vertices, Vertex{
			X: float32FromBytes(vertexData[i:]),
			Y: float32FromBytes(vertexData[i+4:]),
			Z: float32FromBytes(vertexData[i+8:]),
		})
	}
	return nil
}

func (b *BSP) parseVisibility(data []byte) error {
	visibilityData, err := lumpBytes(data, lumpVisibility)
	if err != nil {
		return err
	}
	b.Visibility = visibilityData
	return nil
}

func (b *BSP) parseNodes(data []byte) error {
	nodeData, err := lumpBytes(data, lumpNodes)
	if err != nil {
		return err
	}
	b.Nodes = make([]Node, 0, len(nodeData)/nodeSize)
	for i := 0; i+nodeSize <= len(nodeData); i += nodeSize {
		b.Nodes = append(b.Nodes, Node{
			Plane: int32(binary.LittleEndian.Uint32(nodeData[i:])),
			Children: [2]int16{
				int16(binary.LittleEndian.Uint16(nodeData[i+4:])),
				int16(binary.LittleEndian.Uint16(nodeData[i+6:])),
			},
			Mins: [3]int16{
				int16(binary.LittleEndian.Uint16(nodeData[i+8:])),
				int16(binary.LittleEndian.Uint16(nodeData[i+10:])),
				int16(binary.LittleEndian.Uint16(nodeData[i+12:])),
			},
			Maxs: [3]int16{
				int16(binary.LittleEndian.Uint16(nodeData[i+14:])),
				int16(binary.LittleEndian.Uint16(nodeData[i+16:])),
				int16(binary.LittleEndian.Uint16(nodeData[i+18:])),
			},
			FirstFace: binary.LittleEndian.Uint16(nodeData[i+20:]),
			NumFaces:  binary.LittleEndian.Uint16(nodeData[i+22:]),
		})
	}
	return nil
}

func (b *BSP) parseTexinfo(data []byte) error {
	texinfoData, err := lumpBytes(data, lumpTexinfo)
	if err != nil {
		return err
	}

	b.Texinfo = make([]Texinfo, 0, len(texinfoData)/texinfoSize)
	for texinfoOffset := 0; texinfoOffset+texinfoSize <= len(texinfoData); texinfoOffset += texinfoSize {
		var texinfo Texinfo

		for row := 0; row < 2; row++ {
			for col := 0; col < 4; col++ {
				valueOffset := texinfoOffset + row*16 + col*4
				texinfo.Vecs[row][col] = float32FromBytes(
					texinfoData[valueOffset:],
				)
			}
		}

		texinfo.Miptex = int32(
			binary.LittleEndian.Uint32(texinfoData[texinfoOffset+32:]),
		)
		texinfo.Flags = int32(
			binary.LittleEndian.Uint32(texinfoData[texinfoOffset+36:]),
		)

		b.Texinfo = append(b.Texinfo, texinfo)
	}
	return nil
}

func (b *BSP) parseFaces(data []byte) error {
	faceData, err := lumpBytes(data, lumpFaces)
	if err != nil {
		return err
	}
	b.Faces = make([]Face, 0, len(faceData)/faceSize)
	for i := 0; i+faceSize <= len(faceData); i += faceSize {
		var face Face
		face.Plane = binary.LittleEndian.Uint16(faceData[i:])
		face.Side = binary.LittleEndian.Uint16(faceData[i+2:])
		face.FirstEdge = int32(binary.LittleEndian.Uint32(faceData[i+4:]))
		face.NumEdges = binary.LittleEndian.Uint16(faceData[i+8:])
		face.Texinfo = binary.LittleEndian.Uint16(faceData[i+10:])
		copy(face.Styles[:], faceData[i+12:i+16])
		face.LightOfs = int32(binary.LittleEndian.Uint32(faceData[i+16:]))
		b.Faces = append(b.Faces, face)
	}
	return nil
}

func (b *BSP) parseLighting(data []byte) error {
	lightingData, err := lumpBytes(data, lumpLighting)
	if err != nil {
		return err
	}
	b.Lighting = lightingData
	return nil
}

func (b *BSP) parseClipnodes(data []byte) error {
	clipnodeData, err := lumpBytes(data, lumpClipnodes)
	if err != nil {
		return err
	}
	b.Clipnodes = make([]Clipnode, 0, len(clipnodeData)/clipnodeSize)
	for i := 0; i+clipnodeSize <= len(clipnodeData); i += clipnodeSize {
		b.Clipnodes = append(b.Clipnodes, Clipnode{
			Plane: int32(binary.LittleEndian.Uint32(clipnodeData[i:])),
			Children: [2]int16{
				int16(binary.LittleEndian.Uint16(clipnodeData[i+4:])),
				int16(binary.LittleEndian.Uint16(clipnodeData[i+6:])),
			},
		})
	}
	return nil
}

func (b *BSP) parseLeafs(data []byte) error {
	leafData, err := lumpBytes(data, lumpLeafs)
	if err != nil {
		return err
	}
	b.Leafs = make([]Leaf, 0, len(leafData)/leafSize)
	for i := 0; i+leafSize <= len(leafData); i += leafSize {
		var leaf Leaf
		leaf.Contents = int32(binary.LittleEndian.Uint32(leafData[i:]))
		leaf.VisibilityOffset = int32(binary.LittleEndian.Uint32(leafData[i+4:]))
		leaf.Mins = [3]int16{
			int16(binary.LittleEndian.Uint16(leafData[i+8:])),
			int16(binary.LittleEndian.Uint16(leafData[i+10:])),
			int16(binary.LittleEndian.Uint16(leafData[i+12:])),
		}
		leaf.Maxs = [3]int16{
			int16(binary.LittleEndian.Uint16(leafData[i+14:])),
			int16(binary.LittleEndian.Uint16(leafData[i+16:])),
			int16(binary.LittleEndian.Uint16(leafData[i+18:])),
		}
		leaf.FirstMarkSurface = binary.LittleEndian.Uint16(leafData[i+20:])
		leaf.NumMarkSurfaces = binary.LittleEndian.Uint16(leafData[i+22:])
		copy(leaf.AmbientLevel[:], leafData[i+24:i+28])
		b.Leafs = append(b.Leafs, leaf)
	}
	return nil
}

func (b *BSP) parseMarksurfaces(data []byte) error {
	marksurfaceData, err := lumpBytes(data, lumpMarksurfaces)
	if err != nil {
		return err
	}
	b.Marksurfaces = make([]uint16, 0, len(marksurfaceData)/marksurfaceSize)
	for i := 0; i+marksurfaceSize <= len(marksurfaceData); i += marksurfaceSize {
		b.Marksurfaces = append(
			b.Marksurfaces,
			binary.LittleEndian.Uint16(marksurfaceData[i:]),
		)
	}
	return nil
}

func (b *BSP) parseEdges(data []byte) error {
	edgeData, err := lumpBytes(data, lumpEdges)
	if err != nil {
		return err
	}
	b.Edges = make([]Edge, 0, len(edgeData)/edgeSize)
	for i := 0; i+edgeSize <= len(edgeData); i += edgeSize {
		b.Edges = append(b.Edges, Edge{
			A: binary.LittleEndian.Uint16(edgeData[i:]),
			B: binary.LittleEndian.Uint16(edgeData[i+2:]),
		})
	}
	return nil
}

func (b *BSP) parseSurfedges(data []byte) error {
	surfedgeData, err := lumpBytes(data, lumpSurfedges)
	if err != nil {
		return err
	}
	b.Surfedges = make([]int32, 0, len(surfedgeData)/4)
	for i := 0; i+4 <= len(surfedgeData); i += 4 {
		b.Surfedges = append(
			b.Surfedges,
			int32(binary.LittleEndian.Uint32(surfedgeData[i:])),
		)
	}
	return nil
}

func (b *BSP) parseModels(data []byte) error {
	modelData, err := lumpBytes(data, lumpModels)
	if err != nil {
		return err
	}

	b.Models = make([]Model, 0, len(modelData)/modelSize)
	for modelOffset := 0; modelOffset+modelSize <= len(modelData); modelOffset += modelSize {
		var model Model

		model.Bounds.Min = [3]float32{
			float32FromBytes(modelData[modelOffset:]),
			float32FromBytes(modelData[modelOffset+4:]),
			float32FromBytes(modelData[modelOffset+8:]),
		}
		model.Bounds.Max = [3]float32{
			float32FromBytes(modelData[modelOffset+12:]),
			float32FromBytes(modelData[modelOffset+16:]),
			float32FromBytes(modelData[modelOffset+20:]),
		}
		model.Origin = [3]float32{
			float32FromBytes(modelData[modelOffset+24:]),
			float32FromBytes(modelData[modelOffset+28:]),
			float32FromBytes(modelData[modelOffset+32:]),
		}

		for j := 0; j < 4; j++ {
			model.Headnodes[j] = int32(binary.LittleEndian.Uint32(
				modelData[modelOffset+36+j*4:],
			))
		}

		model.Visleafs = int32(
			binary.LittleEndian.Uint32(modelData[modelOffset+52:]),
		)
		model.FirstFace = int32(
			binary.LittleEndian.Uint32(modelData[modelOffset+56:]),
		)
		model.NumFaces = int32(
			binary.LittleEndian.Uint32(modelData[modelOffset+60:]),
		)

		b.Models = append(b.Models, model)
	}
	return nil
}
