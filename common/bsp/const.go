package bsp

const (
	headerLumpCount = 15

	lumpEntities     = 0
	lumpPlanes       = 1
	lumpMiptex       = 2
	lumpVertices     = 3
	lumpVisibility   = 4
	lumpNodes        = 5
	lumpTexinfo      = 6
	lumpFaces        = 7
	lumpLighting     = 8
	lumpClipnodes    = 9
	lumpLeafs        = 10
	lumpMarksurfaces = 11
	lumpEdges        = 12
	lumpSurfedges    = 13
	lumpModels       = 14

	texSpecial = 1

	planeSize       = 20
	vertexSize      = 12
	nodeSize        = 24
	texinfoSize     = 40
	faceSize        = 20
	clipnodeSize    = 8
	leafSize        = 28
	edgeSize        = 4
	modelSize       = 64
	marksurfaceSize = 2

	versionQuake = 29
)

const (
	contentsEmpty = -1
	contentsSolid = -2
)
