package grib2

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
	"time"
)

// bits packs unsigned values of given widths, big-endian, into octets.
type bits struct {
	out  []byte
	acc  uint64
	used int
}

func (writer *bits) write(value uint64, width int) {
	for bit := width - 1; bit >= 0; bit-- {
		writer.acc = writer.acc<<1 | (value>>bit)&1
		writer.used++
		if writer.used == 8 {
			writer.out = append(writer.out, byte(writer.acc))
			writer.acc, writer.used = 0, 0
		}
	}
}

// align pads to the next whole octet.
func (writer *bits) align() {
	if writer.used > 0 {
		writer.out = append(writer.out, byte(writer.acc<<(8-writer.used)))
		writer.acc, writer.used = 0, 0
	}
}

func u32(value uint32) []byte { return binary.BigEndian.AppendUint32(nil, value) }

// message builds a GRIB2 message on a small latitude/longitude grid from a
// data representation section's body (from octet 6) and section 7's data.
func message(nx, ny int, scan byte, representation, data []byte, bitmap byte) []byte {
	section := func(number byte, body []byte) []byte {
		out := binary.BigEndian.AppendUint32(nil, uint32(5+len(body)))
		return append(append(out, number), body...)
	}
	identification := make([]byte, 16)
	binary.BigEndian.PutUint16(identification[7:], 2026)
	copy(identification[9:], []byte{10, 19, 6, 0, 0})

	grid := []byte{0}
	grid = append(grid, u32(uint32(nx*ny))...)
	grid = append(grid, 0, 0, 0, 0) // no list, template 3.0
	grid = append(grid, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	grid = append(grid, u32(uint32(nx))...)
	grid = append(grid, u32(uint32(ny))...)
	grid = append(grid, u32(0)...)
	grid = append(grid, u32(math.MaxUint32)...)
	grid = append(grid, u32(1<<31|10_000_000)...) // La1 -10°
	grid = append(grid, u32(350_000_000)...)      // Lo1 350°
	grid = append(grid, 0x30)
	grid = append(grid, u32(10_000_000)...)
	grid = append(grid, u32(uint32(350_000_000+(nx-1)*5_000_000)%360_000_000)...)
	grid = append(grid, u32(5_000_000)...)
	grid = append(grid, u32(10_000_000)...)
	grid = append(grid, scan)

	product := []byte{0, 0, 0, 0, 2, 2, 2, 0, 81, 0, 0, 0, 1}
	product = append(product, u32(6)...)
	product = append(product, 103, 0)
	product = append(product, u32(10)...)
	product = append(product, 255, 0, 0, 0, 0, 0)

	body := section(1, identification)
	body = append(body, section(3, grid)...)
	body = append(body, section(4, product)...)
	body = append(body, section(5, representation)...)
	body = append(body, section(6, []byte{bitmap})...)
	body = append(body, section(7, data)...)
	body = append(body, "7777"...)
	out := []byte("GRIB")
	out = append(out, 0, 0, 0, 2)
	out = binary.BigEndian.AppendUint64(out, uint64(16+len(body)))
	return append(out, body...)
}

// simple builds a message with simple packing (template 5.0).
func simple(nx, ny int, scan byte, reference float32, binaryScale, decimalScale uint16, width int, packed []uint64, bitmap byte) []byte {
	representation := u32(uint32(nx * ny))
	representation = append(representation, 0, 0)
	representation = append(representation, u32(math.Float32bits(reference))...)
	representation = binary.BigEndian.AppendUint16(representation, binaryScale)
	representation = binary.BigEndian.AppendUint16(representation, decimalScale)
	representation = append(representation, byte(width), 0)
	var data bits
	for _, value := range packed {
		data.write(value, width)
	}
	data.align()
	return message(nx, ny, scan, representation, data.out, bitmap)
}

func near(t *testing.T, got []float32, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for index := range want {
		if math.Abs(float64(got[index])-want[index]) > 1e-5 {
			t.Fatalf("value %d = %v, want %v (all %v)", index, got[index], want[index], got)
		}
	}
}

func TestDecodeSimplePacking(t *testing.T) {
	// Y = (R + X × 2^E) / 10^D with R = -5, E = 1 and D = 1.
	field, err := Decode(simple(3, 2, 0x40, -5, 1, 1, 6, []uint64{0, 1, 2, 3, 10, 63}, 255))
	if err != nil {
		t.Fatal(err)
	}
	if want := (Grid{Nx: 3, Ny: 2, Lon0: 350, Lat0: -10, DLon: 5, DLat: 10}); field.Grid != want {
		t.Fatalf("grid = %+v, want %+v (scanning south to north)", field.Grid, want)
	}
	if !field.Reference.Equal(time.Date(2026, 10, 19, 6, 0, 0, 0, time.UTC)) || field.ForecastHours != 6 {
		t.Fatalf("reference %v, hours %d", field.Reference, field.ForecastHours)
	}
	if field.Discipline != 0 || field.Category != 2 || field.Number != 2 {
		t.Fatalf("parameter %d.%d.%d", field.Discipline, field.Category, field.Number)
	}
	near(t, field.Values, []float64{-0.5, -0.3, -0.1, 0.1, 1.5, 12.1})

	// A negative binary scale (sign and magnitude) and westward scanning.
	field, err = Decode(simple(2, 1, 0x80, 1, 1<<15|2, 0, 4, []uint64{4, 15}, 255))
	if err != nil {
		t.Fatal(err)
	}
	if field.Grid.DLon != -5 {
		t.Fatalf("dlon = %v", field.Grid.DLon)
	}
	near(t, field.Values, []float64{2, 4.75})

	// No bits per value: a constant field.
	field, err = Decode(simple(2, 2, 0, 3.5, 0, 0, 0, nil, 255))
	if err != nil {
		t.Fatal(err)
	}
	near(t, field.Values, []float64{3.5, 3.5, 3.5, 3.5})
}

// TestDecodeComplexPackingWithSpatialDifferencing builds the packing NOAA's
// GFS files use (template 5.3, first-order differences, two groups) by hand.
func TestDecodeComplexPackingWithSpatialDifferencing(t *testing.T) {
	// Tenths: Y = X / 10 with R = 0, E = 0, D = 1.
	original := []int64{100, 103, 101, 98, 98, 110, 120, 125}
	// First-order differences less their minimum (-3); the first is a
	// placeholder, replaced by the first original value.
	packed := []uint64{0, 6, 1, 0, 3, 15, 13, 8}
	groups := []struct {
		reference uint64
		width     int
		values    []uint64
	}{{0, 3, packed[:4]}, {3, 4, packed[4:]}}

	representation := u32(uint32(len(original)))
	representation = append(representation, 0, 3) // template 5.3
	representation = append(representation, u32(math.Float32bits(0))...)
	representation = binary.BigEndian.AppendUint16(representation, 0) // E
	representation = binary.BigEndian.AppendUint16(representation, 1) // D
	representation = append(representation, 4, 0)                     // group references in 4 bits; floating point
	representation = append(representation, 1, 0)                     // general group splitting; no missing values
	representation = append(representation, u32(0)...)
	representation = append(representation, u32(0)...)
	representation = append(representation, u32(uint32(len(groups)))...) // NG
	representation = append(representation, 0, 4)                        // width reference 0, 4 bits
	representation = append(representation, u32(4)...)                   // length reference 4
	representation = append(representation, 1)                           // length increment
	representation = append(representation, u32(4)...)                   // last group's true length
	representation = append(representation, 1)                           // bits per scaled length
	representation = append(representation, 1, 2)                        // first order, 2-octet descriptors

	var data bits
	data.write(100, 16)     // first original value
	data.write(1<<15|3, 16) // minimum difference, -3
	for _, group := range groups {
		data.write(group.reference, 4)
	}
	data.align()
	for _, group := range groups {
		data.write(uint64(group.width), 4)
	}
	data.align()
	for range groups {
		data.write(0, 1) // both lengths 4 + 0 × 1
	}
	data.align()
	for _, group := range groups {
		for _, value := range group.values {
			data.write(value-group.reference, group.width)
		}
	}
	data.align()

	field, err := Decode(message(4, 2, 0x40, representation, data.out, 255))
	if err != nil {
		t.Fatal(err)
	}
	want := make([]float64, len(original))
	for index, value := range original {
		want[index] = float64(value) / 10
	}
	near(t, field.Values, want)
}

func TestDecodeRefusesWhatItCannotRead(t *testing.T) {
	if _, err := Decode(simple(2, 1, 0, 0, 0, 0, 8, []uint64{1, 2}, 0)); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("a bitmap: %v", err)
	}
	if _, err := Decode(simple(2, 1, 0x20, 0, 0, 0, 8, []uint64{1, 2}, 255)); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("columns stored first: %v", err)
	}
	raw := simple(2, 1, 0, 0, 0, 0, 8, []uint64{1, 2}, 255)
	if _, err := Decode(raw[:len(raw)-6]); err == nil {
		t.Fatal("a truncated message must fail")
	}
	if _, err := Decode(simple(4, 1, 0, 0, 0, 0, 8, []uint64{1, 2}, 255)); !errors.Is(err, errShortData) {
		t.Fatalf("short data: %v", err)
	}
	if _, err := Decode([]byte("<html>Not Found</html>")); err == nil {
		t.Fatal("HTML must fail")
	}
	for cut := 20; cut < len(raw)-4; cut += 7 {
		if _, err := Decode(append([]byte(nil), raw[:cut]...)); err == nil {
			t.Fatalf("cut at %d must fail", cut)
		}
	}
}
