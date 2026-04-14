package bsp

func (b *BSP) Polygons() []Polygon {
	polygons := make([]Polygon, 0, len(b.Faces))

	for _, face := range b.Faces {
		texinfoIndex := int(face.Texinfo)
		if texinfoIndex < 0 || texinfoIndex >= len(b.Texinfo) {
			continue
		}

		vertices := make([]Vertex, 0, int(face.NumEdges))
		firstEdge := int(face.FirstEdge)

		for edgeIndex := 0; edgeIndex < int(face.NumEdges); edgeIndex++ {
			surfedgeIndex := b.Surfedges[firstEdge+edgeIndex]
			var vertexIndex uint16
			if surfedgeIndex >= 0 {
				vertexIndex = b.Edges[surfedgeIndex].A
			} else {
				vertexIndex = b.Edges[-surfedgeIndex].B
			}
			if int(vertexIndex) >= len(b.Vertices) {
				continue
			}
			vertices = append(vertices, b.Vertices[vertexIndex])
		}

		if len(vertices) < 3 {
			continue
		}
		polygons = append(polygons, Polygon{
			Vertices: vertices,
			Special:  b.Texinfo[texinfoIndex].Flags&texSpecial != 0,
		})
	}

	return polygons
}
