package geo

import (
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

func TestSimplifyLineKeepsEndsAndSignificantPoints(t *testing.T) {
	line := []Point{{0, 0}, {1, 0.00001}, {2, 0}, {3, 1}, {4, 0}}
	got := SimplifyLine(line, 0.001)
	want := []Point{{0, 0}, {2, 0}, {3, 1}, {4, 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SimplifyLine = %v, want %v", got, want)
	}
	if got := SimplifyLine(line, 0); !reflect.DeepEqual(got, line) {
		t.Fatalf("zero tolerance changed the line: %v", got)
	}
}

func TestSimplifyLineSplitsClosedLines(t *testing.T) {
	ring := []Point{{0, 0}, {1, 0}, {2, 0.00001}, {2, 2}, {0, 2}, {0, 0}}
	got := SimplifyLine(ring, 0.001)
	if got[0] != got[len(got)-1] {
		t.Fatalf("closed line no longer closed: %v", got)
	}
	if len(got) != 5 {
		t.Fatalf("closed line = %v, want the four corners closed", got)
	}
}

func TestWKBEncodesMultiPolygon(t *testing.T) {
	shape := MultiPolygon{{{{0, 0}, {1, 0}, {1, 1}, {0, 0}}}}
	wkb := shape.WKB()
	if len(wkb) != 1+4+4+1+4+4+4+4*16 {
		t.Fatalf("WKB length = %d", len(wkb))
	}
	if wkb[0] != 1 || binary.LittleEndian.Uint32(wkb[1:]) != 6 || binary.LittleEndian.Uint32(wkb[5:]) != 1 {
		t.Fatalf("WKB header = %v", wkb[:9])
	}
	if binary.LittleEndian.Uint32(wkb[10:]) != 3 || binary.LittleEndian.Uint32(wkb[18:]) != 4 {
		t.Fatalf("WKB polygon header = %v", wkb[9:22])
	}
	if x := math.Float64frombits(binary.LittleEndian.Uint64(wkb[22+16:])); x != 1 {
		t.Fatalf("second point x = %v, want 1", x)
	}
}

func TestBoundsAndPoints(t *testing.T) {
	shape := MultiPolygon{
		{{{0, 0}, {2, 0}, {2, 1}, {0, 0}}},
		{{{-1, 5}, {0, 5}, {0, 6}, {-1, 5}}},
	}
	if got := shape.Bounds(); got != (Bounds{West: -1, South: 0, East: 2, North: 6}) {
		t.Fatalf("Bounds = %+v", got)
	}
	if shape.Points() != 8 {
		t.Fatalf("Points = %d, want 8", shape.Points())
	}
}

// twoSquares is a quantized topology of two unit squares sharing one wavy
// edge (arc 0), plus a feature without geometry.
const twoSquares = `{
  "type": "Topology",
  "transform": {"scale": [0.001, 0.001], "translate": [80, 7]},
  "objects": {"regions": {"type": "GeometryCollection", "geometries": [
    {"type": "Polygon", "arcs": [[0, 1]], "properties": {"id": "west"}},
    {"type": "MultiPolygon", "arcs": [[[2, -1]]], "properties": {"id": "east"}},
    {"type": null, "properties": {"id": "none"}}
  ]}},
  "arcs": [
    [[1000, 0], [1, 500], [-1, 500]],
    [[1000, 1000], [-1000, 0], [0, -1000], [1000, 0]],
    [[1000, 0], [1000, 0], [0, 1000], [-1000, 0]]
  ]
}`

func TestTopologyFeatures(t *testing.T) {
	topology, err := DecodeTopology([]byte(twoSquares))
	if err != nil {
		t.Fatal(err)
	}
	if got := topology.Objects(); !reflect.DeepEqual(got, []string{"regions"}) {
		t.Fatalf("Objects = %v", got)
	}
	features, skipped, err := topology.Features("regions", 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if len(features) != 2 || skipped != 1 {
		t.Fatalf("features = %d, skipped = %d; want 2 and 1", len(features), skipped)
	}
	west := features[0].Shape[0][0]
	if west[0] != (Point{81, 7}) || west[len(west)-1] != west[0] {
		t.Fatalf("west ring = %v, want a closed ring from (81, 7)", west)
	}
	if len(west) != 6 {
		t.Fatalf("west ring has %d points, want 6 (shared point not repeated)", len(west))
	}
	// The shared edge's middle point is within the tolerance, so both
	// simplified squares drop it and still meet along the same edge.
	westSimple, eastSimple := features[0].Simple[0][0], features[1].Simple[0][0]
	if len(westSimple) != 5 || len(eastSimple) != 5 {
		t.Fatalf("simplified rings = %v and %v, want squares", westSimple, eastSimple)
	}
	if !contains(westSimple, Point{81, 8}) || !contains(eastSimple, Point{81, 8}) {
		t.Fatalf("simplified squares do not share the corner (81, 8): %v %v", westSimple, eastSimple)
	}
	if string(features[1].Properties) != `{"id": "east"}` {
		t.Fatalf("properties = %s", features[1].Properties)
	}
}

func TestTopologyRejectsBadInput(t *testing.T) {
	if _, err := DecodeTopology([]byte(`{"type": "FeatureCollection"}`)); err == nil {
		t.Fatal("expected an error for a non-topology document")
	}
	topology, err := DecodeTopology([]byte(`{"type": "Topology", "objects": {"a": {"geometries": [{"type": "Polygon", "arcs": [[5]]}]}}, "arcs": []}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := topology.Features("a", 0); err == nil {
		t.Fatal("expected an error for an arc index out of range")
	}
	if _, _, err := topology.Features("missing", 0); err == nil {
		t.Fatal("expected an error for a missing object")
	}
}

func contains(ring Ring, point Point) bool {
	for _, candidate := range ring {
		if math.Abs(candidate[0]-point[0]) < 1e-9 && math.Abs(candidate[1]-point[1]) < 1e-9 {
			return true
		}
	}
	return false
}
