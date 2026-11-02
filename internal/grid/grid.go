// Package grid defines the regular longitude/latitude grid over Sri Lanka on
// which weather is collected, and reads values between its nodes.
//
// The grid is a lattice of nodes Step degrees apart. Only nodes near land are
// kept (the land mask): a node is kept when any of the four lattice cells it
// is a corner of touches land, so every point on land has all four
// surrounding nodes and bilinear interpolation never has to guess at the
// coast. The kept sea nodes also carry the near-shore wind and rain.
package grid

import (
	"math"
)

// Lattice is a regular grid: node (i, j) lies at Lon0 + i×Step east and
// Lat0 + j×Step north, and has index j×Nx + i. Rows run south to north.
type Lattice struct {
	Lon0 float64 `json:"lon0"`
	Lat0 float64 `json:"lat0"`
	Step float64 `json:"step"`
	Nx   int     `json:"nx"`
	Ny   int     `json:"ny"`
}

// Lon is the longitude of column i.
func (lattice Lattice) Lon(i int) float64 { return round6(lattice.Lon0 + float64(i)*lattice.Step) }

// Lat is the latitude of row j.
func (lattice Lattice) Lat(j int) float64 { return round6(lattice.Lat0 + float64(j)*lattice.Step) }

// Index is the index of node (i, j).
func (lattice Lattice) Index(i, j int) int { return j*lattice.Nx + i }

// Size is the number of nodes in the lattice.
func (lattice Lattice) Size() int { return lattice.Nx * lattice.Ny }

// Contains reports whether a point lies within the lattice's outer nodes.
func (lattice Lattice) Contains(lon, lat float64) bool {
	column := (lon - lattice.Lon0) / lattice.Step
	row := (lat - lattice.Lat0) / lattice.Step
	return column >= 0 && column <= float64(lattice.Nx-1) && row >= 0 && row <= float64(lattice.Ny-1)
}

// Node is one kept node.
type Node struct {
	Index int     `json:"index"`
	I     int     `json:"i"`
	J     int     `json:"j"`
	Lon   float64 `json:"lon"`
	Lat   float64 `json:"lat"`
	// Land is whether the node itself is on land; the source is asked for a
	// land model cell there and a sea cell otherwise.
	Land bool `json:"land"`
}

// Grid is a lattice with its land mask.
type Grid struct {
	ID string
	Lattice
	// Nodes are the kept nodes in index order.
	Nodes    []Node
	position []int
}

// SriLankaID names the default grid; stored runs record the grid they used.
const SriLankaID = "lk-0.2"

// SriLankaLattice is 0.2° from 79.4° E, 5.6° N to 82.2° E, 10.0° N: Sri
// Lanka (79.65 to 81.89° E, 5.92 to 9.83° N) with a cell to spare on every
// side. See docs/decisions/0007-weather-grid-and-flood-forecast.md for why
// 0.2° and not 0.1°.
var SriLankaLattice = Lattice{Lon0: 79.4, Lat0: 5.6, Step: 0.2, Nx: 15, Ny: 23}

// SriLanka returns the default weather grid.
func SriLanka() Grid { return Build(SriLankaID, SriLankaLattice) }

// Coarser grids for air quality and sea state, which their models do not
// resolve as finely as the weather grid (ADR 0012).
const (
	AirQualityID = "lk-air-0.4"
	SeaID        = "lk-sea-0.5"
)

// AirQualityLattice is 0.4°, the CAMS global model's own resolution, over the
// weather grid's extent.
var AirQualityLattice = Lattice{Lon0: 79.4, Lat0: 5.6, Step: 0.4, Nx: 8, Ny: 12}

// SeaLattice is 0.5° from 79° E, 5° N to 82.5° E, 10.5° N: the waters within
// about 70 km of Sri Lanka's coast, which is what coastal users need.
var SeaLattice = Lattice{Lon0: 79, Lat0: 5, Step: 0.5, Nx: 8, Ny: 12}

// AirQuality returns the air-quality grid, masked like the weather grid.
func AirQuality() Grid { return Build(AirQualityID, AirQualityLattice) }

// Sea returns the sea-state grid: the lattice's nodes that are not on land.
func Sea() Grid {
	sea := Grid{ID: SeaID, Lattice: SeaLattice, position: make([]int, SeaLattice.Size())}
	for j := range SeaLattice.Ny {
		for i := range SeaLattice.Nx {
			index := SeaLattice.Index(i, j)
			sea.position[index] = -1
			lon, lat := SeaLattice.Lon(i), SeaLattice.Lat(j)
			if OnLand(lon, lat) {
				continue
			}
			sea.position[index] = len(sea.Nodes)
			sea.Nodes = append(sea.Nodes, Node{Index: index, I: i, J: j, Lon: lon, Lat: lat})
		}
	}
	return sea
}

// NearSea reports whether a point is on the water, or within `reach` degrees of
// it in any of eight directions.
func NearSea(lon, lat, reach float64) bool {
	if !OnLand(lon, lat) {
		return true
	}
	const diagonal = 0.7071
	for _, step := range [][2]float64{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {diagonal, diagonal}, {diagonal, -diagonal}, {-diagonal, diagonal}, {-diagonal, -diagonal}} {
		if !OnLand(lon+step[0]*reach, lat+step[1]*reach) {
			return true
		}
	}
	return false
}

// Land returns Sri Lanka's land as closed outer rings of longitude, latitude
// pairs, for masking a sea field to the coast.
func Land() [][][2]float64 { return coastline }

// Build applies the land mask to a lattice.
func Build(id string, lattice Lattice) Grid {
	grid := Grid{ID: id, Lattice: lattice, position: make([]int, lattice.Size())}
	reach := lattice.Step
	for j := range lattice.Ny {
		for i := range lattice.Nx {
			index := lattice.Index(i, j)
			grid.position[index] = -1
			lon, lat := lattice.Lon(i), lattice.Lat(j)
			// The four cells around the node together span one step each way.
			if !touchesLand(lon-reach, lat-reach, lon+reach, lat+reach) {
				continue
			}
			grid.position[index] = len(grid.Nodes)
			grid.Nodes = append(grid.Nodes, Node{Index: index, I: i, J: j, Lon: lon, Lat: lat, Land: OnLand(lon, lat)})
		}
	}
	return grid
}

// Position returns where a lattice node sits in Nodes, if it is kept.
func (grid Grid) Position(index int) (int, bool) {
	if index < 0 || index >= len(grid.position) || grid.position[index] < 0 {
		return 0, false
	}
	return grid.position[index], true
}

// Expand spreads values given per kept node, in node order, over the whole
// lattice, with NaN at masked nodes. Missing values must already be NaN.
func (grid Grid) Expand(values []float32) []float32 {
	out := make([]float32, grid.Size())
	for index := range out {
		out[index] = float32(math.NaN())
	}
	for position, node := range grid.Nodes {
		if position < len(values) {
			out[node.Index] = values[position]
		}
	}
	return out
}

// Weight is one node's share in an interpolated value.
type Weight struct {
	Index  int
	Weight float64
}

// Corners returns the four nodes around a point with their bilinear
// weights, or false when the point is outside the lattice.
func (lattice Lattice) Corners(lon, lat float64) ([4]Weight, bool) {
	if !lattice.Contains(lon, lat) {
		return [4]Weight{}, false
	}
	column := (lon - lattice.Lon0) / lattice.Step
	row := (lat - lattice.Lat0) / lattice.Step
	x0, y0 := int(math.Floor(column)), int(math.Floor(row))
	x1, y1 := min(x0+1, lattice.Nx-1), min(y0+1, lattice.Ny-1)
	tx, ty := column-float64(x0), row-float64(y0)
	return [4]Weight{
		{Index: lattice.Index(x0, y0), Weight: (1 - tx) * (1 - ty)},
		{Index: lattice.Index(x1, y0), Weight: tx * (1 - ty)},
		{Index: lattice.Index(x0, y1), Weight: (1 - tx) * ty},
		{Index: lattice.Index(x1, y1), Weight: tx * ty},
	}, true
}

// Sample is the bilinear value at the corners' point from values laid out
// over the whole lattice. Corners without a value (NaN) are left out and the
// others' weights scaled up, so a point beside a masked node still gets the
// nearest values; false when no corner with weight has a value.
func Sample(values []float32, corners [4]Weight) (float64, bool) {
	var sum, weights float64
	for _, corner := range corners {
		if corner.Weight <= 0 || corner.Index >= len(values) {
			continue
		}
		value := float64(values[corner.Index])
		if math.IsNaN(value) {
			continue
		}
		sum += value * corner.Weight
		weights += corner.Weight
	}
	if weights == 0 {
		return 0, false
	}
	return sum / weights, true
}

// Nearest is the value of the corner with the largest weight that has one,
// for categories such as weather codes, which cannot be averaged.
func Nearest(values []float32, corners [4]Weight) (float64, bool) {
	best, found := -1.0, false
	var value float64
	for _, corner := range corners {
		if corner.Index >= len(values) || math.IsNaN(float64(values[corner.Index])) {
			continue
		}
		if corner.Weight > best {
			best, value, found = corner.Weight, float64(values[corner.Index]), true
		}
	}
	return value, found
}

// OnLand reports whether a point is on Sri Lanka's land.
func OnLand(lon, lat float64) bool {
	for _, ring := range coastline {
		if inRing(lon, lat, ring) {
			return true
		}
	}
	return false
}

// touchesLand reports whether a box intersects land: a corner of the box is
// on land, a vertex of the coast is in the box, or an edge of the box crosses
// the coast.
func touchesLand(west, south, east, north float64) bool {
	corners := [5][2]float64{{west, south}, {east, south}, {east, north}, {west, north}, {west, south}}
	for _, corner := range corners[:4] {
		if OnLand(corner[0], corner[1]) {
			return true
		}
	}
	for _, ring := range coastline {
		for index, vertex := range ring {
			if vertex[0] >= west && vertex[0] <= east && vertex[1] >= south && vertex[1] <= north {
				return true
			}
			if index == 0 {
				continue
			}
			for side := range 4 {
				if segmentsCross(ring[index-1], vertex, corners[side], corners[side+1]) {
					return true
				}
			}
		}
	}
	return false
}

// inRing is the even-odd rule.
func inRing(lon, lat float64, ring [][2]float64) bool {
	inside := false
	for index := 1; index < len(ring); index++ {
		a, b := ring[index-1], ring[index]
		if (a[1] > lat) != (b[1] > lat) && lon < a[0]+(lat-a[1])*(b[0]-a[0])/(b[1]-a[1]) {
			inside = !inside
		}
	}
	return inside
}

func segmentsCross(a, b, c, d [2]float64) bool {
	cross := func(o, p, q [2]float64) float64 {
		return (p[0]-o[0])*(q[1]-o[1]) - (p[1]-o[1])*(q[0]-o[0])
	}
	d1, d2 := cross(c, d, a), cross(c, d, b)
	d3, d4 := cross(a, b, c), cross(a, b, d)
	return ((d1 > 0) != (d2 > 0)) && ((d3 > 0) != (d4 > 0))
}

func round6(value float64) float64 { return math.Round(value*1e6) / 1e6 }
