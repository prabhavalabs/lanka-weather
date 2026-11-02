package geo

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
)

// signedArea is a ring's area in square degrees by the shoelace formula:
// positive when counter-clockwise, negative when clockwise.
func signedArea(ring Ring) float64 {
	area := 0.0
	for i := 1; i < len(ring); i++ {
		area += ring[i-1][0]*ring[i][1] - ring[i][0]*ring[i-1][1]
	}
	return area / 2
}

// ringContains reports whether a point lies inside a ring, by ray casting.
func ringContains(ring Ring, point Point) bool {
	inside := false
	for i, j := 0, len(ring)-1; i < len(ring); j, i = i, i+1 {
		a, b := ring[i], ring[j]
		if (a[1] > point[1]) != (b[1] > point[1]) &&
			point[0] < (b[0]-a[0])*(point[1]-a[1])/(b[1]-a[1])+a[0] {
			inside = !inside
		}
	}
	return inside
}

// PolygonFromRings groups the rings of a shapefile or ArcGIS polygon, where
// outer rings run clockwise and holes counter-clockwise. A hole goes to the
// smallest outer ring containing it; a hole inside nothing is kept as an
// outer ring, because mislabelled rings are common and losing land is worse
// than filling a hole. Rings with fewer than four points are dropped.
func PolygonFromRings(rings []Ring) MultiPolygon {
	type outer struct {
		ring Ring
		area float64
	}
	var outers []outer
	var holes []Ring
	for _, ring := range rings {
		if len(ring) < 4 {
			continue
		}
		if signedArea(ring) > 0 {
			holes = append(holes, ring)
			continue
		}
		outers = append(outers, outer{ring: ring, area: math.Abs(signedArea(ring))})
	}
	sort.SliceStable(outers, func(i, j int) bool { return outers[i].area < outers[j].area })
	shape := make(MultiPolygon, len(outers))
	for i, candidate := range outers {
		shape[i] = Polygon{candidate.ring}
	}
	for _, hole := range holes {
		placed := false
		for i, candidate := range outers {
			if ringContains(candidate.ring, hole[0]) {
				shape[i] = append(shape[i], hole)
				placed = true
				break
			}
		}
		if !placed {
			shape = append(shape, Polygon{hole})
		}
	}
	return shape
}

// LineWKB encodes a line as little-endian well-known binary.
func LineWKB(line []Point) []byte {
	var buffer bytes.Buffer
	buffer.Grow(9 + len(line)*16)
	const (
		littleEndian = 1
		wkbLine      = 2
	)
	buffer.WriteByte(littleEndian)
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(wkbLine))
	_ = binary.Write(&buffer, binary.LittleEndian, uint32(len(line)))
	for _, point := range line {
		_ = binary.Write(&buffer, binary.LittleEndian, point[0])
		_ = binary.Write(&buffer, binary.LittleEndian, point[1])
	}
	return buffer.Bytes()
}
