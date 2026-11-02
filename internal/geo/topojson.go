package geo

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Topology is a decoded TopoJSON topology. Neighbouring polygons share arcs,
// so simplifying the arcs once simplifies every shared border the same way
// and the simplified polygons still meet without gaps or overlaps.
type Topology struct {
	arcs    [][]Point
	objects map[string]topoCollection
}

// Feature is one polygon feature of a topology.
type Feature struct {
	// Properties are the feature's properties as published.
	Properties json.RawMessage
	Shape      MultiPolygon
	// Simple is Shape simplified along its arcs; it equals Shape when no
	// tolerance was given.
	Simple MultiPolygon
}

type topoDocument struct {
	Type      string `json:"type"`
	Transform *struct {
		Scale     [2]float64 `json:"scale"`
		Translate [2]float64 `json:"translate"`
	} `json:"transform"`
	Arcs    [][][]float64             `json:"arcs"`
	Objects map[string]topoCollection `json:"objects"`
}

type topoCollection struct {
	Type       string         `json:"type"`
	Geometries []topoGeometry `json:"geometries"`
}

type topoGeometry struct {
	Type       string          `json:"type"`
	Arcs       json.RawMessage `json:"arcs"`
	Properties json.RawMessage `json:"properties"`
}

// DecodeTopology parses a TopoJSON document, decoding quantized arcs.
func DecodeTopology(payload []byte) (*Topology, error) {
	var document topoDocument
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, fmt.Errorf("decode topology: %w", err)
	}
	if document.Type != "Topology" {
		return nil, fmt.Errorf("decode topology: type is %q, want Topology", document.Type)
	}
	topology := &Topology{arcs: make([][]Point, len(document.Arcs)), objects: document.Objects}
	for index, arc := range document.Arcs {
		points := make([]Point, len(arc))
		var x, y float64
		for i, position := range arc {
			if len(position) < 2 {
				return nil, fmt.Errorf("decode topology: arc %d has a position with %d values", index, len(position))
			}
			if document.Transform == nil {
				points[i] = Point{position[0], position[1]}
				continue
			}
			// Quantized arcs are delta-encoded integers.
			x += position[0]
			y += position[1]
			points[i] = Point{
				x*document.Transform.Scale[0] + document.Transform.Translate[0],
				y*document.Transform.Scale[1] + document.Transform.Translate[1],
			}
		}
		topology.arcs[index] = points
	}
	return topology, nil
}

// Objects returns the names of the topology's objects.
func (topology *Topology) Objects() []string {
	names := make([]string, 0, len(topology.objects))
	for name := range topology.objects {
		names = append(names, name)
	}
	return names
}

// Features returns the polygon features of an object. Each feature's Simple
// shape has its arcs simplified with the given tolerance in degrees; a ring
// that would collapse keeps its full detail. Features without a polygon
// geometry are skipped and counted.
func (topology *Topology) Features(object string, tolerance float64) (features []Feature, skipped int, err error) {
	collection, ok := topology.objects[object]
	if !ok {
		return nil, 0, fmt.Errorf("topology has no object %q", object)
	}
	simple := topology.arcs
	if tolerance > 0 {
		simple = make([][]Point, len(topology.arcs))
		for index, arc := range topology.arcs {
			simple[index] = SimplifyLine(arc, tolerance)
		}
	}
	for index, geometry := range collection.Geometries {
		var rings [][][]int
		switch geometry.Type {
		case "Polygon":
			var polygon [][]int
			if err := json.Unmarshal(geometry.Arcs, &polygon); err != nil {
				return nil, 0, fmt.Errorf("geometry %d: %w", index, err)
			}
			rings = [][][]int{polygon}
		case "MultiPolygon":
			if err := json.Unmarshal(geometry.Arcs, &rings); err != nil {
				return nil, 0, fmt.Errorf("geometry %d: %w", index, err)
			}
		default:
			skipped++
			continue
		}
		shape, simplified, err := assemble(rings, topology.arcs, simple)
		if err != nil {
			return nil, 0, fmt.Errorf("geometry %d: %w", index, err)
		}
		if len(shape) == 0 {
			skipped++
			continue
		}
		features = append(features, Feature{Properties: geometry.Properties, Shape: shape, Simple: simplified})
	}
	return features, skipped, nil
}

var errBadArc = errors.New("arc index out of range")

// assemble joins arcs into rings, once from the full arcs and once from the
// simplified ones. A simplified ring that collapses keeps its full detail; a
// polygon whose full outer ring is degenerate is dropped.
func assemble(polygons [][][]int, full, simple [][]Point) (shape, simplified MultiPolygon, err error) {
	for _, polygon := range polygons {
		var rings, simpleRings Polygon
		for r, indexes := range polygon {
			ring, err := joinArcs(indexes, full)
			if err != nil {
				return nil, nil, err
			}
			if len(ring) < 4 {
				if r == 0 {
					break
				}
				continue
			}
			simpleRing, err := joinArcs(indexes, simple)
			if err != nil {
				return nil, nil, err
			}
			if len(simpleRing) < 4 {
				simpleRing = ring
			}
			rings = append(rings, ring)
			simpleRings = append(simpleRings, simpleRing)
		}
		if len(rings) > 0 {
			shape = append(shape, rings)
			simplified = append(simplified, simpleRings)
		}
	}
	return shape, simplified, nil
}

// joinArcs builds a closed ring from arc indexes; a negative index ~i is
// arc i reversed.
func joinArcs(indexes []int, arcs [][]Point) (Ring, error) {
	var ring Ring
	for _, index := range indexes {
		reversed := index < 0
		if reversed {
			index = ^index
		}
		if index >= len(arcs) {
			return nil, errBadArc
		}
		arc := arcs[index]
		if len(ring) > 0 && len(arc) > 0 {
			// Consecutive arcs share their joining point.
			ring = ring[:len(ring)-1]
		}
		if reversed {
			for i := len(arc) - 1; i >= 0; i-- {
				ring = append(ring, arc[i])
			}
		} else {
			ring = append(ring, arc...)
		}
	}
	if len(ring) > 0 && ring[0] != ring[len(ring)-1] {
		ring = append(ring, ring[0])
	}
	return ring, nil
}
