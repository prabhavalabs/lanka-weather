// Package geo holds the small amount of planar geometry the collectors need
// before handing shapes to PostGIS: polygon types, TopoJSON decoding,
// simplification along shared edges and well-known binary encoding.
//
// Coordinates are longitude, latitude in degrees (EPSG:4326).
package geo

import (
	"bytes"
	"encoding/binary"
	"math"
)

// Point is a longitude, latitude pair.
type Point [2]float64

// Ring is a closed ring: its first and last points are equal.
type Ring []Point

// Polygon is an outer ring followed by any holes.
type Polygon []Ring

// MultiPolygon is one or more polygons.
type MultiPolygon []Polygon

// Bounds is a bounding box in degrees.
type Bounds struct {
	West, South, East, North float64
}

// Bounds returns the box around every point of the shape. The box of an
// empty shape has West > East.
func (shape MultiPolygon) Bounds() Bounds {
	box := Bounds{West: math.Inf(1), South: math.Inf(1), East: math.Inf(-1), North: math.Inf(-1)}
	for _, polygon := range shape {
		for _, ring := range polygon {
			for _, point := range ring {
				box.West = math.Min(box.West, point[0])
				box.East = math.Max(box.East, point[0])
				box.South = math.Min(box.South, point[1])
				box.North = math.Max(box.North, point[1])
			}
		}
	}
	return box
}

// Points counts the shape's vertices.
func (shape MultiPolygon) Points() int {
	count := 0
	for _, polygon := range shape {
		for _, ring := range polygon {
			count += len(ring)
		}
	}
	return count
}

// WKB encodes the shape as little-endian well-known binary, which PostGIS
// reads with ST_GeomFromWKB.
func (shape MultiPolygon) WKB() []byte {
	var buffer bytes.Buffer
	buffer.Grow(9 + shape.Points()*16 + len(shape)*9)
	write := func(value any) { _ = binary.Write(&buffer, binary.LittleEndian, value) }
	const (
		littleEndian = 1
		wkbPolygon   = 3
		wkbMulti     = 6
	)
	buffer.WriteByte(littleEndian)
	write(uint32(wkbMulti))
	write(uint32(len(shape)))
	for _, polygon := range shape {
		buffer.WriteByte(littleEndian)
		write(uint32(wkbPolygon))
		write(uint32(len(polygon)))
		for _, ring := range polygon {
			write(uint32(len(ring)))
			for _, point := range ring {
				write(point[0])
				write(point[1])
			}
		}
	}
	return buffer.Bytes()
}

// SimplifyLine reduces a line with the Douglas-Peucker algorithm, keeping
// every point that is more than tolerance degrees from the simplified line.
// The first and last points are always kept, so lines that meet still meet.
func SimplifyLine(line []Point, tolerance float64) []Point {
	if len(line) <= 2 || tolerance <= 0 {
		return append([]Point(nil), line...)
	}
	// A closed line (an island drawn as one arc) has no baseline; split it at
	// the point farthest from its start and simplify each half.
	if line[0] == line[len(line)-1] {
		farthest, distance := 0, -1.0
		for index, point := range line {
			if d := squaredDistance(point, line[0]); d > distance {
				farthest, distance = index, d
			}
		}
		if farthest == 0 {
			return append([]Point(nil), line...)
		}
		first := SimplifyLine(line[:farthest+1], tolerance)
		second := SimplifyLine(line[farthest:], tolerance)
		return append(first, second[1:]...)
	}
	keep := make([]bool, len(line))
	keep[0], keep[len(line)-1] = true, true
	limit := tolerance * tolerance
	stack := [][2]int{{0, len(line) - 1}}
	for len(stack) > 0 {
		span := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		start, end := span[0], span[1]
		farthest, distance := -1, limit
		for index := start + 1; index < end; index++ {
			if d := segmentDistance(line[index], line[start], line[end]); d > distance {
				farthest, distance = index, d
			}
		}
		if farthest >= 0 {
			keep[farthest] = true
			stack = append(stack, [2]int{start, farthest}, [2]int{farthest, end})
		}
	}
	out := make([]Point, 0, len(line)/4+2)
	for index, point := range line {
		if keep[index] {
			out = append(out, point)
		}
	}
	return out
}

func squaredDistance(a, b Point) float64 {
	dx, dy := a[0]-b[0], a[1]-b[1]
	return dx*dx + dy*dy
}

// segmentDistance is the squared distance from p to the segment a-b.
func segmentDistance(p, a, b Point) float64 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	if dx == 0 && dy == 0 {
		return squaredDistance(p, a)
	}
	t := ((p[0]-a[0])*dx + (p[1]-a[1])*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return squaredDistance(p, Point{a[0] + t*dx, a[1] + t*dy})
}
