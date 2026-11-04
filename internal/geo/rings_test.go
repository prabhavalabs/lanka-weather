package geo

import "testing"

func square(west, south, size float64, clockwise bool) Ring {
	ring := Ring{{west, south}, {west, south + size}, {west + size, south + size}, {west + size, south}, {west, south}}
	if clockwise {
		return ring
	}
	reversed := make(Ring, len(ring))
	for i, point := range ring {
		reversed[len(ring)-1-i] = point
	}
	return reversed
}

func TestPolygonFromRingsPutsHolesInTheirOuterRing(t *testing.T) {
	shape := PolygonFromRings([]Ring{
		square(0, 0, 10, true),
		square(20, 0, 10, true),
		square(21, 1, 2, false), // a hole in the second square
	})
	if len(shape) != 2 {
		t.Fatalf("%d polygons, want 2", len(shape))
	}
	var withHole int
	for _, polygon := range shape {
		if len(polygon) == 2 {
			withHole++
			if polygon[0][0][0] != 20 {
				t.Fatalf("hole went to the outer ring starting at %v", polygon[0][0])
			}
		}
	}
	if withHole != 1 {
		t.Fatalf("%d polygons with a hole, want 1", withHole)
	}
}

func TestPolygonFromRingsKeepsAnOrphanHoleAsLand(t *testing.T) {
	shape := PolygonFromRings([]Ring{square(0, 0, 10, false)})
	if len(shape) != 1 || len(shape[0]) != 1 {
		t.Fatalf("a counter-clockwise ring alone should stay as land, got %v", shape)
	}
}

func TestPolygonFromRingsDropsDegenerateRings(t *testing.T) {
	if shape := PolygonFromRings([]Ring{{{0, 0}, {1, 1}, {0, 0}}}); len(shape) != 0 {
		t.Fatalf("got %v", shape)
	}
}

func TestLineWKBLength(t *testing.T) {
	if got := len(LineWKB([]Point{{1, 2}, {3, 4}})); got != 9+32 {
		t.Fatalf("length %d", got)
	}
}
