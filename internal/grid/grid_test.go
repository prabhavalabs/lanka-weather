package grid

import (
	"math"
	"testing"
)

func TestOnLand(t *testing.T) {
	land := map[string][2]float64{
		"Kandy": {80.6337, 7.2906}, "Colombo": {79.8612, 6.9271}, "Jaffna": {80.0255, 9.6615},
		"Trincomalee": {81.2152, 8.5874}, "Hambantota": {81.1185, 6.1241},
	}
	for name, point := range land {
		if !OnLand(point[0], point[1]) {
			t.Errorf("%s should be on land", name)
		}
	}
	sea := map[string][2]float64{"Gulf of Mannar": {79.5, 8.5}, "south of Dondra": {80.6, 5.7}, "Bay of Bengal": {82.0, 8.0}}
	for name, point := range sea {
		if OnLand(point[0], point[1]) {
			t.Errorf("%s should be at sea", name)
		}
	}
}

func TestSriLankaGridKeepsNodesNearLand(t *testing.T) {
	grid := SriLanka()
	if n := len(grid.Nodes); n < 150 || n > 260 {
		t.Fatalf("kept %d nodes; the 0.2° grid should keep about 200 of %d", n, grid.Size())
	}
	land := 0
	for position, node := range grid.Nodes {
		if got, ok := grid.Position(node.Index); !ok || got != position {
			t.Fatalf("node %d position = %d %v", node.Index, got, ok)
		}
		if position > 0 && grid.Nodes[position-1].Index >= node.Index {
			t.Fatal("nodes must be in index order")
		}
		if node.Land {
			land++
		}
	}
	if land == 0 || land == len(grid.Nodes) {
		t.Fatalf("land nodes = %d of %d; expected both land and near-shore sea nodes", land, len(grid.Nodes))
	}
	// The lattice's corners are open sea, far from land.
	for _, index := range []int{0, grid.Nx - 1, grid.Index(0, grid.Ny-1), grid.Size() - 1} {
		if _, ok := grid.Position(index); ok {
			t.Fatalf("corner node %d should be masked", index)
		}
	}
}

// Every point on land must have all four surrounding nodes, so interpolation
// on land never falls back to fewer corners.
func TestEveryLandPointHasFourNodes(t *testing.T) {
	grid := SriLanka()
	checked := 0
	for lat := 5.9; lat <= 9.9; lat += 0.013 {
		for lon := 79.6; lon <= 81.95; lon += 0.013 {
			if !OnLand(lon, lat) {
				continue
			}
			checked++
			corners, ok := grid.Corners(lon, lat)
			if !ok {
				t.Fatalf("%.3f, %.3f is outside the lattice", lon, lat)
			}
			for _, corner := range corners {
				if _, kept := grid.Position(corner.Index); !kept {
					t.Fatalf("%.3f, %.3f: corner %d is masked", lon, lat, corner.Index)
				}
			}
		}
	}
	if checked < 1000 {
		t.Fatalf("only %d land points checked", checked)
	}
}

func TestSampleReproducesAPlane(t *testing.T) {
	lattice := Lattice{Lon0: 80, Lat0: 6, Step: 0.5, Nx: 3, Ny: 3}
	values := make([]float32, lattice.Size())
	for j := range lattice.Ny {
		for i := range lattice.Nx {
			values[lattice.Index(i, j)] = float32(10 + 2*lattice.Lon(i) - 3*lattice.Lat(j))
		}
	}
	for _, point := range [][2]float64{{80.2, 6.7}, {80, 6}, {81, 7}, {80.75, 6.25}} {
		corners, ok := lattice.Corners(point[0], point[1])
		if !ok {
			t.Fatalf("%v outside", point)
		}
		got, ok := Sample(values, corners)
		want := 10 + 2*point[0] - 3*point[1]
		if !ok || math.Abs(got-want) > 1e-4 {
			t.Fatalf("Sample(%v) = %v, want %v", point, got, want)
		}
	}
	if _, ok := lattice.Corners(79.9, 6.5); ok {
		t.Fatal("a point west of the lattice must be outside")
	}
}

func TestSampleSkipsMissingCorners(t *testing.T) {
	lattice := Lattice{Lon0: 0, Lat0: 0, Step: 1, Nx: 2, Ny: 2}
	nan := float32(math.NaN())
	values := []float32{10, 20, nan, 40}
	corners, _ := lattice.Corners(0.5, 0.5)
	got, ok := Sample(values, corners)
	if !ok || math.Abs(got-70.0/3) > 1e-6 {
		t.Fatalf("Sample = %v %v, want the mean of the three present corners", got, ok)
	}
	if _, ok := Sample([]float32{nan, nan, nan, nan}, corners); ok {
		t.Fatal("no present corner must give no value")
	}
	corners, _ = lattice.Corners(0.1, 0.8)
	if got, ok := Nearest(values, corners); !ok || got != 10 {
		// (0, 1) is nearest but missing; (0, 0) has the next largest weight.
		t.Fatalf("Nearest = %v %v, want 10", got, ok)
	}
}

func TestExpand(t *testing.T) {
	grid := SriLanka()
	values := make([]float32, len(grid.Nodes))
	for i := range values {
		values[i] = float32(i)
	}
	full := grid.Expand(values)
	if len(full) != grid.Size() {
		t.Fatalf("len = %d", len(full))
	}
	node := grid.Nodes[7]
	if full[node.Index] != 7 || !math.IsNaN(float64(full[0])) {
		t.Fatalf("expand put %v at node 7 and %v at a masked node", full[node.Index], full[0])
	}
}
