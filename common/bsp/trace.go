package bsp

func (b *BSP) HasLineOfSight(from, to [3]float64) bool {
	rootNodeIndex, ok := b.rootNodeIndex()
	if !ok {
		return false
	}

	return b.segmentIsVisible(rootNodeIndex, from, to)
}

func (b *BSP) segmentIsVisible(
	nodeIndex int,
	from [3]float64,
	to [3]float64,
) bool {
	const planeEpsilon = 0.03125

	if nodeIndex < 0 {
		return b.leafIsVisible(-nodeIndex - 1)
	}

	if nodeIndex >= len(b.Nodes) {
		return false
	}

	node := b.Nodes[nodeIndex]
	if node.Plane < 0 || int(node.Plane) >= len(b.Planes) {
		return false
	}

	plane := b.Planes[node.Plane]
	fromDistance := planeDistance(plane, from)
	toDistance := planeDistance(plane, to)

	if fromDistance >= 0 && toDistance >= 0 {
		return b.segmentIsVisible(int(node.Children[0]), from, to)
	}

	if fromDistance < 0 && toDistance < 0 {
		return b.segmentIsVisible(int(node.Children[1]), from, to)
	}

	firstSide := 0
	nearFraction := 0.0
	farFraction := 1.0

	if fromDistance < toDistance {
		firstSide = 1
		nearFraction = (fromDistance + planeEpsilon) / (fromDistance - toDistance)
		farFraction = (fromDistance - planeEpsilon) / (fromDistance - toDistance)
	} else if fromDistance > toDistance {
		firstSide = 0
		nearFraction = (fromDistance - planeEpsilon) / (fromDistance - toDistance)
		farFraction = (fromDistance + planeEpsilon) / (fromDistance - toDistance)
	}

	nearFraction = clampFraction(nearFraction)
	farFraction = clampFraction(farFraction)

	nearPoint := pointAtFraction(from, to, nearFraction)
	farPoint := pointAtFraction(from, to, farFraction)

	if !b.segmentIsVisible(int(node.Children[firstSide]), from, nearPoint) {
		return false
	}

	return b.segmentIsVisible(int(node.Children[firstSide^1]), farPoint, to)
}

func (b *BSP) rootNodeIndex() (int, bool) {
	if len(b.Models) == 0 {
		return 0, len(b.Nodes) > 0
	}

	if b.Models[0].Headnodes[0] < 0 {
		return 0, false
	}

	return int(b.Models[0].Headnodes[0]), true
}

func (b *BSP) leafIsVisible(leafIndex int) bool {
	if leafIndex < 0 || leafIndex >= len(b.Leafs) {
		return false
	}

	return b.Leafs[leafIndex].Contents != contentsSolid
}

func planeDistance(plane Plane, point [3]float64) float64 {
	return float64(plane.Normal[0])*point[0] +
		float64(plane.Normal[1])*point[1] +
		float64(plane.Normal[2])*point[2] -
		float64(plane.Dist)
}

func pointAtFraction(from, to [3]float64, fraction float64) [3]float64 {
	return [3]float64{
		from[0] + (to[0]-from[0])*fraction,
		from[1] + (to[1]-from[1])*fraction,
		from[2] + (to[2]-from[2])*fraction,
	}
}

func clampFraction(fraction float64) float64 {
	if fraction < 0 {
		return 0
	}

	if fraction > 1 {
		return 1
	}

	return fraction
}
