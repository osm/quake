package bsp

import "strings"

type BSP struct {
	Version int32
	offsets [headerLumpCount]int

	Entities     []Entity
	Planes       []Plane
	Miptex       []byte
	Vertices     []Vertex
	Visibility   []byte
	Nodes        []Node
	Texinfo      []Texinfo
	Faces        []Face
	Lighting     []byte
	Clipnodes    []Clipnode
	Leafs        []Leaf
	Marksurfaces []uint16
	Edges        []Edge
	Surfedges    []int32
	Models       []Model
}

type Bounds struct {
	Min [3]float32
	Max [3]float32
}

type Vertex struct {
	X float32
	Y float32
	Z float32
}

type Plane struct {
	Normal [3]float32
	Dist   float32
	Type   int32
}

type Node struct {
	Plane     int32
	Mins      [3]int16
	Maxs      [3]int16
	Children  [2]int16
	FirstFace uint16
	NumFaces  uint16
}

type Texinfo struct {
	Vecs   [2][4]float32
	Miptex int32
	Flags  int32
}

type Face struct {
	Plane     uint16
	Side      uint16
	FirstEdge int32
	NumEdges  uint16
	Texinfo   uint16
	Styles    [4]byte
	LightOfs  int32
}

type Edge struct {
	A uint16
	B uint16
}

type Clipnode struct {
	Plane    int32
	Children [2]int16
}

type Leaf struct {
	Contents         int32
	VisibilityOffset int32
	Mins             [3]int16
	Maxs             [3]int16
	FirstMarkSurface uint16
	NumMarkSurfaces  uint16
	AmbientLevel     [4]byte
}

type Model struct {
	Bounds    Bounds
	Origin    [3]float32
	Headnodes [4]int32
	Visleafs  int32
	FirstFace int32
	NumFaces  int32
}

type Entity struct {
	Index     int
	Classname string
	Pairs     []EntityField
	Origin    [3]float64
}

type EntityField struct {
	Key   string
	Value string
}

type Polygon struct {
	Vertices []Vertex
	Special  bool
}

func (e *Entity) Value(key string) string {
	for _, field := range e.Pairs {
		if field.Key == key {
			return field.Value
		}
	}
	return ""
}

func (e *Entity) SetValue(key, value string) {
	for i := range e.Pairs {
		if e.Pairs[i].Key != key {
			continue
		}
		e.Pairs[i].Value = value
		e.refreshDerived()
		return
	}
	e.Pairs = append(e.Pairs, EntityField{Key: key, Value: value})
	e.refreshDerived()
}

func (e *Entity) refreshDerived() {
	e.Classname = strings.ToLower(strings.TrimSpace(e.Value("classname")))
	e.Origin = parseEntityOrigin(e.Value("origin"))
}
