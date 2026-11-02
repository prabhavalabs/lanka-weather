// Package grib2 decodes one GRIB edition 2 message (WMO FM 92, the Manual on
// Codes, volume I.2): the subset the wind layer reads (ADR 0007; adopted from TerraGuard). That is a
// regular latitude/longitude grid (grid definition template 3.0) without a
// bitmap, packed simply (data representation template 5.0), with complex
// packing (5.2) or with complex packing and spatial differencing (5.3, which
// NOAA's GFS files use). Anything else is refused with an error rather than
// misread.
//
// A message is a run of sections, each starting with its length (4 octets)
// and number (1 octet): 0 indicator, 1 identification, 2 local use,
// 3 grid, 4 product, 5 data representation, 6 bitmap, 7 data and 8 "7777".
// Octet numbers in comments are the specification's, counted from 1 at the
// start of each section.
package grib2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

// maxPoints bounds the grids decoded, so a corrupt header cannot ask for
// gigabytes. The 0.25° global grid has about a million points.
const maxPoints = 4 << 20

// ErrUnsupported is wrapped by every error about a valid message this
// package does not read: another grid, packing or a bitmap.
var ErrUnsupported = errors.New("unsupported GRIB2 message")

// Grid is a regular latitude/longitude grid in the order the values are
// stored: value j*Nx+i lies at Lat0+j*DLat, Lon0+i*DLon degrees. A negative
// DLat runs north to south, as GFS does from 90° N.
type Grid struct {
	Nx   int     `json:"nx"`
	Ny   int     `json:"ny"`
	Lon0 float64 `json:"lon0"`
	Lat0 float64 `json:"lat0"`
	DLon float64 `json:"dlon"`
	DLat float64 `json:"dlat"`
}

// Field is one decoded message.
type Field struct {
	// Discipline, Category and Number name the parameter (code tables 0.0,
	// 4.1 and 4.2): 0, 2, 2 is the u-component of wind and 0, 2, 3 the v.
	Discipline, Category, Number int
	// Surface is the first fixed surface's type (code table 4.5; 103 is a
	// height above ground) and SurfaceValue its value (10 for 10 m).
	Surface      int
	SurfaceValue float64
	// Reference is the model run's time, and ForecastHours the lead time
	// when the product states it in hours.
	Reference     time.Time
	ForecastHours int
	Grid          Grid
	Values        []float32
}

// Decode reads one GRIB2 message.
func Decode(message []byte) (Field, error) {
	if len(message) < 16 || string(message[:4]) != "GRIB" {
		return Field{}, errors.New("not a GRIB message")
	}
	if message[7] != 2 {
		return Field{}, fmt.Errorf("%w: edition %d", ErrUnsupported, message[7])
	}
	total := binary.BigEndian.Uint64(message[8:16])
	if total > uint64(len(message)) {
		return Field{}, fmt.Errorf("message is %d bytes, %d given", total, len(message))
	}
	field := Field{Discipline: int(message[6])}
	var representation, data []byte
	points, bitmap := -1, -1
	for offset := 16; ; {
		if offset+4 <= int(total) && string(message[offset:offset+4]) == "7777" {
			break
		}
		if offset+5 > int(total) {
			return Field{}, errors.New("message ends without 7777")
		}
		length := int(binary.BigEndian.Uint32(message[offset:]))
		if length < 5 || offset+length > int(total) {
			return Field{}, fmt.Errorf("section at %d has length %d", offset, length)
		}
		section := message[offset : offset+length]
		var err error
		switch section[4] {
		case 1:
			err = field.readIdentification(section)
		case 3:
			points, err = field.readGrid(section)
		case 4:
			field.readProduct(section)
		case 5:
			representation = section
		case 6:
			if len(section) < 6 {
				return Field{}, errors.New("short bitmap section")
			}
			bitmap = int(section[5])
		case 7:
			data = section[5:]
		}
		if err != nil {
			return Field{}, err
		}
		offset += length
	}
	switch {
	case points < 0:
		return Field{}, errors.New("message has no grid")
	case representation == nil || data == nil:
		return Field{}, errors.New("message has no data")
	case bitmap != 255:
		return Field{}, fmt.Errorf("%w: bitmap indicator %d", ErrUnsupported, bitmap)
	}
	values, err := unpack(representation, data, points)
	if err != nil {
		return Field{}, err
	}
	field.Values = values
	return field, nil
}

// readIdentification reads section 1's reference time (octets 13–19).
func (field *Field) readIdentification(section []byte) error {
	if len(section) < 19 {
		return errors.New("short identification section")
	}
	year := int(binary.BigEndian.Uint16(section[12:14]))
	field.Reference = time.Date(year, time.Month(section[14]), int(section[15]), int(section[16]), int(section[17]), int(section[18]), 0, time.UTC)
	return nil
}

// readGrid reads section 3 with grid definition template 3.0 and returns the
// number of data points.
func (field *Field) readGrid(section []byte) (int, error) {
	if len(section) < 14 {
		return 0, errors.New("short grid section")
	}
	points := int(binary.BigEndian.Uint32(section[6:10]))
	if section[10] != 0 {
		return 0, fmt.Errorf("%w: a list of points per row", ErrUnsupported)
	}
	if template := binary.BigEndian.Uint16(section[12:14]); template != 0 {
		return 0, fmt.Errorf("%w: grid template 3.%d", ErrUnsupported, template)
	}
	if len(section) < 72 {
		return 0, errors.New("short latitude/longitude grid")
	}
	word := func(octet int) uint32 { return binary.BigEndian.Uint32(section[octet-1 : octet+3]) }
	nx, ny := int(word(31)), int(word(35))
	if nx < 1 || ny < 1 || nx*ny != points || points > maxPoints {
		return 0, fmt.Errorf("grid of %d × %d for %d points", nx, ny, points)
	}
	// Angles are in millionths of a degree unless a basic angle and its
	// subdivisions say otherwise.
	unit := 1e-6
	if basic, subdivisions := word(39), word(43); basic != 0 && basic != math.MaxUint32 && subdivisions != 0 && subdivisions != math.MaxUint32 {
		unit = float64(basic) / float64(subdivisions)
	}
	angle := func(octet int) float64 { return float64(signMagnitude(uint64(word(octet)), 32)) * unit }
	lat1, lon1, lat2, lon2 := angle(47), angle(51), angle(56), angle(60)
	di, dj := float64(word(64))*unit, float64(word(68))*unit
	// Increments may be left out (flag bits 3 and 4 of octet 55): take them from the corners.
	if section[54]&0x20 == 0 || word(64) == math.MaxUint32 {
		di = math.Abs(normalise(lon2-lon1)) / float64(max(nx-1, 1))
	}
	if section[54]&0x10 == 0 || word(68) == math.MaxUint32 {
		dj = math.Abs(lat2-lat1) / float64(max(ny-1, 1))
	}
	scan := section[71]
	if scan&0x30 != 0 {
		return 0, fmt.Errorf("%w: scanning mode %08b", ErrUnsupported, scan)
	}
	grid := Grid{Nx: nx, Ny: ny, Lon0: lon1, Lat0: lat1, DLon: di, DLat: -dj}
	if scan&0x80 != 0 {
		grid.DLon = -di
	}
	if scan&0x40 != 0 {
		grid.DLat = dj
	}
	field.Grid = grid
	return points, nil
}

// readProduct reads the parameter from section 4 and, for templates laid out
// like 4.0, the first fixed surface and the forecast time in hours.
func (field *Field) readProduct(section []byte) {
	if len(section) < 11 {
		return
	}
	field.Category, field.Number = int(section[9]), int(section[10])
	switch binary.BigEndian.Uint16(section[7:9]) {
	case 0, 1, 2, 8, 11, 12:
	default:
		return
	}
	if len(section) < 28 {
		return
	}
	if section[17] == 1 { // forecast time unit: hour
		field.ForecastHours = int(signMagnitude(uint64(binary.BigEndian.Uint32(section[18:22])), 32))
	}
	field.Surface = int(section[22])
	scale := signMagnitude(uint64(section[23]), 8)
	field.SurfaceValue = float64(binary.BigEndian.Uint32(section[24:28])) * math.Pow10(-int(scale))
}

// normalise wraps a longitude difference into [0, 360).
func normalise(degrees float64) float64 {
	degrees = math.Mod(degrees, 360)
	if degrees < 0 {
		degrees += 360
	}
	return degrees
}

// signMagnitude reads GRIB's signed integers: the top bit is the sign, the
// rest the magnitude (not two's complement).
func signMagnitude(value uint64, bits int) int64 {
	sign := uint64(1) << (bits - 1)
	if value&sign != 0 {
		return -int64(value &^ sign)
	}
	return int64(value)
}
