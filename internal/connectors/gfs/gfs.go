// Package gfs collects the 10 m wind from NOAA's Global Forecast System for
// the seas around Sri Lanka, for the map's animated wind layer.
//
// GFS runs four times a day and publishes each run's 0.25° grids on NOAA
// Open Data on AWS as one GRIB2 file per forecast hour, with an inventory
// (.idx) beside it; both are public and keyless. The collector reads only
// the two wind messages of each file by byte range (about 1 MB each of a
// 500 MB file), crops them to the region and stores them. The approach and
// the decoder are adopted from TerraGuard (ADR 0007).
package gfs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grib2"
)

// SourceName identifies the source in stored records and docs/data-sources.md.
const SourceName = "noaa-gfs"

// Attribution is the credit shown with the layer. GFS is a US government
// work in the public domain.
const Attribution = "NOAA GFS"

const (
	// DefaultBaseURL is the bucket on NOAA Open Data on AWS.
	DefaultBaseURL = "https://noaa-gfs-bdp-pds.s3.amazonaws.com"
	// runEvery is the time between runs (00, 06, 12 and 18 UTC).
	runEvery = 6 * time.Hour
	// publishDelay is how long a run takes to appear; younger runs are not
	// asked for. Its last hour we read is usually there 4 to 5 hours after
	// the run time.
	publishDelay = 3 * time.Hour
	// runLookback is how far back runs are tried.
	runLookback = 24 * time.Hour
	// Step is the time between the forecast hours stored, and MaxForecast
	// the furthest ahead: three days, the span of the hourly forecast.
	Step        = 3 * time.Hour
	MaxForecast = 72 * time.Hour
	// Model and Level name the field.
	Model = "GFS 0.25°"
	Level = "10 m above ground"
	// Scale is the m/s of one stored unit: GFS packs its wind to a tenth of
	// a metre per second, so int16 tenths keep everything it has.
	Scale = 0.1
	// maxMessageBytes bounds one message read; a 0.25° wind message is
	// about 1 MB.
	maxMessageBytes = 8 << 20
)

// Region is the box stored: the map's panning limit, wide enough to follow
// a cyclone approaching from the Bay of Bengal or the Arabian Sea.
var Region = Box{West: 70, South: -2, East: 92, North: 18}

// Box is a longitude/latitude box.
type Box struct{ West, South, East, North float64 }

// Hours lists the forecast hours stored for each run.
func Hours() []int {
	var hours []int
	for hour := time.Duration(0); hour <= MaxForecast; hour += Step {
		hours = append(hours, int(hour/time.Hour))
	}
	return hours
}

// Runs lists the runs that may be published by now, newest first.
func Runs(now time.Time) []time.Time {
	var runs []time.Time
	newest := now.UTC().Add(-publishDelay).Truncate(runEvery)
	for run := newest; !run.Before(now.UTC().Add(-runLookback)); run = run.Add(-runEvery) {
		runs = append(runs, run)
	}
	return runs
}

// FileURL is the GRIB2 file of one run and forecast hour on the 0.25° grid.
func FileURL(base string, run time.Time, hour int) string {
	return fmt.Sprintf("%s/gfs.%s/%02d/atmos/gfs.t%02dz.pgrb2.0p25.f%03d", base, run.Format("20060102"), run.Hour(), run.Hour(), hour)
}

// indexEntry is one line of a GRIB2 inventory:
// "588:422979508:d=2026110206:UGRD:10 m above ground:72 hour fcst:".
type indexEntry struct {
	Offset   int64
	Date     string
	Variable string
	Level    string
}

// ParseIndex reads an inventory. Lines it cannot read are skipped, but an
// answer with none (an error page, say) is an error.
func ParseIndex(text []byte) ([]indexEntry, error) {
	var entries []indexEntry
	for line := range bytes.Lines(text) {
		parts := strings.Split(strings.TrimRight(string(line), "\r\n"), ":")
		if len(parts) < 6 || !strings.HasPrefix(parts[2], "d=") {
			continue
		}
		if _, err := strconv.ParseFloat(parts[0], 64); err != nil {
			continue
		}
		offset, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || offset < 0 {
			continue
		}
		entries = append(entries, indexEntry{Offset: offset, Date: strings.TrimPrefix(parts[2], "d="), Variable: parts[3], Level: parts[4]})
	}
	if len(entries) == 0 {
		return nil, errors.New("not a GRIB2 inventory")
	}
	return entries, nil
}

// ByteRange is a span of a file, both ends included, as an HTTP Range
// header gives it.
type ByteRange struct{ Start, End int64 }

// Header is the Range header for the span.
func (span ByteRange) Header() string { return fmt.Sprintf("bytes=%d-%d", span.Start, span.End) }

// Size is the span's length in bytes.
func (span ByteRange) Size() int64 { return span.End - span.Start + 1 }

// MessageRange finds one message of a run's inventory. It runs from its
// offset to the byte before the next message's.
func MessageRange(entries []indexEntry, run time.Time, variable, level string) (ByteRange, error) {
	date := run.Format("2006010215")
	for index, entry := range entries {
		if entry.Variable != variable || entry.Level != level {
			continue
		}
		if entry.Date != date {
			return ByteRange{}, fmt.Errorf("inventory is for run %s, not %s", entry.Date, date)
		}
		for _, next := range entries[index+1:] {
			if next.Offset > entry.Offset {
				return ByteRange{Start: entry.Offset, End: next.Offset - 1}, nil
			}
		}
		return ByteRange{}, fmt.Errorf("%s at %s is the file's last message", variable, level)
	}
	return ByteRange{}, fmt.Errorf("inventory has no %s at %s", variable, level)
}

// Crop cuts the box out of a field on a regular grid, keeping the source's
// row order, and returns the cropped grid and values. Longitudes may be given
// from -180 or from 0; the box must not cross the grid's seam.
func Crop(grid grib2.Grid, values []float32, box Box) (grib2.Grid, []float32, error) {
	if len(values) != grid.Nx*grid.Ny || grid.DLon <= 0 || grid.DLat == 0 {
		return grib2.Grid{}, nil, errors.New("field does not match its grid")
	}
	column := func(lon float64) float64 {
		offset := math.Mod(lon-grid.Lon0+720, 360)
		return offset / grid.DLon
	}
	i0, i1 := int(math.Floor(column(box.West)+1e-9)), int(math.Ceil(column(box.East)-1e-9))
	if i1 < i0 || i1 >= grid.Nx {
		return grib2.Grid{}, nil, fmt.Errorf("box %v crosses the grid's edge", box)
	}
	rowOf := func(lat float64) float64 { return (lat - grid.Lat0) / grid.DLat }
	j0, j1 := rowOf(box.South), rowOf(box.North)
	if j0 > j1 {
		j0, j1 = j1, j0
	}
	r0, r1 := int(math.Floor(j0+1e-9)), int(math.Ceil(j1-1e-9))
	if r0 < 0 || r1 >= grid.Ny {
		return grib2.Grid{}, nil, fmt.Errorf("box %v is outside the grid", box)
	}
	out := grib2.Grid{
		Nx: i1 - i0 + 1, Ny: r1 - r0 + 1, DLon: grid.DLon, DLat: grid.DLat,
		Lon0: normaliseLon(grid.Lon0 + float64(i0)*grid.DLon), Lat0: grid.Lat0 + float64(r0)*grid.DLat,
	}
	cropped := make([]float32, 0, out.Nx*out.Ny)
	for row := r0; row <= r1; row++ {
		cropped = append(cropped, values[row*grid.Nx+i0:row*grid.Nx+i1+1]...)
	}
	return out, cropped, nil
}

// normaliseLon brings a longitude into -180 to 180.
func normaliseLon(lon float64) float64 {
	lon = math.Mod(lon+540, 360) - 180
	return math.Round(lon*1e6) / 1e6
}

// Quantise packs values as little-endian int16s of Scale.
func Quantise(values []float32) []byte {
	raw := make([]byte, 2*len(values))
	for index, value := range values {
		units := math.Round(float64(value) / Scale)
		if math.IsNaN(units) {
			units = 0
		}
		units = math.Max(math.MinInt16, math.Min(math.MaxInt16, units))
		binary.LittleEndian.PutUint16(raw[2*index:], uint16(int16(units)))
	}
	return raw
}

// Dequantise reverses Quantise.
func Dequantise(raw []byte) []float32 {
	values := make([]float32, len(raw)/2)
	for index := range values {
		values[index] = float32(float64(int16(binary.LittleEndian.Uint16(raw[2*index:]))) * Scale)
	}
	return values
}
