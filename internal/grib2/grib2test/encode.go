// Package grib2test builds small GRIB2 messages for tests, so tests need no
// downloaded GRIB files (which are never committed).
package grib2test

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grib2"
)

// Message describes one message to encode.
type Message struct {
	Grid      grib2.Grid
	Reference time.Time
	Hour      int
	// Category and Number name the parameter in discipline 0: 2, 2 is the
	// u-component of wind and 2, 3 the v-component.
	Category, Number int
	// Values in grid order, stored to a tenth (decimal scale 1).
	Values []float32
}

// Encode writes the message with a latitude/longitude grid (template 3.0),
// a product at 10 m above ground (4.0) and simple packing (5.0) in 16 bits.
func Encode(message Message) []byte {
	u32 := func(value uint32) []byte { return binary.BigEndian.AppendUint32(nil, value) }
	angle := func(degrees float64) uint32 {
		micro := uint32(math.Round(math.Abs(degrees) * 1e6))
		if degrees < 0 {
			micro |= 1 << 31
		}
		return micro
	}
	section := func(number byte, body []byte) []byte {
		out := binary.BigEndian.AppendUint32(nil, uint32(5+len(body)))
		return append(append(out, number), body...)
	}
	grid := message.Grid
	points := grid.Nx * grid.Ny

	identification := make([]byte, 16)
	at := message.Reference.UTC()
	binary.BigEndian.PutUint16(identification[7:], uint16(at.Year()))
	copy(identification[9:], []byte{byte(at.Month()), byte(at.Day()), byte(at.Hour()), byte(at.Minute()), byte(at.Second())})

	var scan byte
	if grid.DLon < 0 {
		scan |= 0x80
	}
	if grid.DLat > 0 {
		scan |= 0x40
	}
	gridSection := []byte{0}
	gridSection = append(gridSection, u32(uint32(points))...)
	gridSection = append(gridSection, 0, 0, 0, 0)
	gridSection = append(gridSection, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	gridSection = append(gridSection, u32(uint32(grid.Nx))...)
	gridSection = append(gridSection, u32(uint32(grid.Ny))...)
	gridSection = append(gridSection, u32(0)...)
	gridSection = append(gridSection, u32(math.MaxUint32)...)
	gridSection = append(gridSection, u32(angle(grid.Lat0))...)
	gridSection = append(gridSection, u32(angle(grid.Lon0))...)
	gridSection = append(gridSection, 0x30)
	gridSection = append(gridSection, u32(angle(grid.Lat0+float64(grid.Ny-1)*grid.DLat))...)
	gridSection = append(gridSection, u32(angle(grid.Lon0+float64(grid.Nx-1)*grid.DLon))...)
	gridSection = append(gridSection, u32(angle(math.Abs(grid.DLon)))...)
	gridSection = append(gridSection, u32(angle(math.Abs(grid.DLat)))...)
	gridSection = append(gridSection, scan)

	product := []byte{0, 0, 0, 0, byte(message.Category), byte(message.Number), 2, 0, 81, 0, 0, 0, 1}
	product = append(product, u32(uint32(message.Hour))...)
	product = append(product, 103, 0)
	product = append(product, u32(10)...)
	product = append(product, 255, 0, 0, 0, 0, 0)

	reference := math.Inf(1)
	for _, value := range message.Values {
		reference = math.Min(reference, math.Round(float64(value)*10))
	}
	if math.IsInf(reference, 1) {
		reference = 0
	}
	representation := u32(uint32(points))
	representation = append(representation, 0, 0)
	representation = append(representation, u32(math.Float32bits(float32(reference)))...)
	representation = binary.BigEndian.AppendUint16(representation, 0) // E
	representation = binary.BigEndian.AppendUint16(representation, 1) // D: tenths
	representation = append(representation, 16, 0)
	data := make([]byte, 0, 2*points)
	for _, value := range message.Values {
		data = binary.BigEndian.AppendUint16(data, uint16(math.Round(float64(value)*10)-reference))
	}

	body := section(1, identification)
	body = append(body, section(3, gridSection)...)
	body = append(body, section(4, product)...)
	body = append(body, section(5, representation)...)
	body = append(body, section(6, []byte{255})...)
	body = append(body, section(7, data)...)
	body = append(body, "7777"...)
	out := []byte("GRIB")
	out = append(out, 0, 0, 0, 2)
	out = binary.BigEndian.AppendUint64(out, uint64(16+len(body)))
	return append(out, body...)
}
