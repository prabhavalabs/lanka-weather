package gfs

import (
	"math"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grib2"
	"github.com/prabhavalabs/lanka-weather/internal/grib2/grib2test"
)

func TestHoursAndRuns(t *testing.T) {
	hours := Hours()
	if len(hours) != 25 || hours[0] != 0 || hours[1] != 3 || hours[24] != 72 {
		t.Fatalf("hours = %v", hours)
	}
	runs := Runs(time.Date(2026, 11, 2, 10, 15, 0, 0, time.UTC))
	// 10:15 less the publication delay is 07:15: the 06 UTC run is the newest asked for.
	if len(runs) != 4 || !runs[0].Equal(time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)) ||
		!runs[3].Equal(time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("runs = %v", runs)
	}
	if got := FileURL("https://bucket", runs[0], 72); got != "https://bucket/gfs.20261102/06/atmos/gfs.t06z.pgrb2.0p25.f072" {
		t.Fatalf("FileURL = %s", got)
	}
}

func TestIndexAndMessageRange(t *testing.T) {
	index := []byte("1:0:d=2026110206:PRMSL:mean sea level:72 hour fcst:\n" +
		"garbage line\n" +
		"588:422979508:d=2026110206:UGRD:10 m above ground:72 hour fcst:\n" +
		"589:423937111:d=2026110206:VGRD:10 m above ground:72 hour fcst:\n" +
		"590:424872422:d=2026110206:ICEG:10 m above mean sea level:72 hour fcst:\n")
	entries, err := ParseIndex(index)
	if err != nil || len(entries) != 4 {
		t.Fatalf("entries = %v, %v", entries, err)
	}
	run := time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)
	span, err := MessageRange(entries, run, "VGRD", Level)
	if err != nil || span.Header() != "bytes=423937111-424872421" || span.Size() != 935311 {
		t.Fatalf("span = %+v %v", span, err)
	}
	if _, err := MessageRange(entries, run.Add(-6*time.Hour), "UGRD", Level); err == nil {
		t.Fatal("an inventory of another run must be refused")
	}
	if _, err := MessageRange(entries, run, "ICEG", "10 m above mean sea level"); err == nil {
		t.Fatal("the last message has no known end")
	}
	if _, err := ParseIndex([]byte("<Error>NoSuchKey</Error>")); err == nil {
		t.Fatal("an error page is not an inventory")
	}
}

func TestCropFromAGlobalNorthToSouthGrid(t *testing.T) {
	// A 1° global grid like GFS's: from 90° N southward, from 0° E eastward.
	global := grib2.Grid{Nx: 360, Ny: 181, Lon0: 0, Lat0: 90, DLon: 1, DLat: -1}
	values := make([]float32, global.Nx*global.Ny)
	for j := range global.Ny {
		for i := range global.Nx {
			values[j*global.Nx+i] = float32(1000*j + i) // row j is latitude 90 - j
		}
	}
	grid, cropped, err := Crop(global, values, Region)
	if err != nil {
		t.Fatal(err)
	}
	want := grib2.Grid{Nx: 23, Ny: 21, Lon0: 70, Lat0: 18, DLon: 1, DLat: -1}
	if grid != want || len(cropped) != 23*21 {
		t.Fatalf("grid = %+v (%d values), want %+v", grid, len(cropped), want)
	}
	// The first value is 18° N 70° E; the last 2° S 92° E.
	if cropped[0] != 1000*72+70 || cropped[len(cropped)-1] != 1000*92+92 {
		t.Fatalf("corners = %v, %v", cropped[0], cropped[len(cropped)-1])
	}
	// A grid given from -180° is cropped the same way.
	shifted := grib2.Grid{Nx: 360, Ny: 181, Lon0: -180, Lat0: 90, DLon: 1, DLat: -1}
	grid, _, err = Crop(shifted, values, Box{West: -10, South: 0, East: 10, North: 10})
	if err != nil || grid.Lon0 != -10 || grid.Nx != 21 {
		t.Fatalf("shifted crop = %+v %v", grid, err)
	}
	if _, _, err := Crop(global, values, Box{West: -10, South: 0, East: 10, North: 10}); err == nil {
		t.Fatal("a box across the grid's seam must be refused")
	}
}

func TestQuantiseRoundTrip(t *testing.T) {
	values := []float32{0, 0.04, -12.35, 25.5, float32(math.NaN()), 5000}
	back := Dequantise(Quantise(values))
	want := []float32{0, 0, -12.4, 25.5, 0, 3276.7}
	for index := range want {
		if math.Abs(float64(back[index]-want[index])) > 0.051 {
			t.Fatalf("value %d: %v came back as %v", index, values[index], back[index])
		}
	}
}

func TestEncodedMessageDecodesAndCrops(t *testing.T) {
	run := time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)
	grid := grib2.Grid{Nx: 30, Ny: 25, Lon0: 66, Lat0: 21, DLon: 1, DLat: -1}
	values := make([]float32, grid.Nx*grid.Ny)
	for index := range values {
		values[index] = float32(index%7) - 3.3
	}
	field, err := grib2.Decode(grib2test.Encode(grib2test.Message{Grid: grid, Reference: run, Hour: 9, Category: 2, Number: 3, Values: values}))
	if err != nil {
		t.Fatal(err)
	}
	if field.Grid != grid || field.Number != 3 || field.ForecastHours != 9 || !field.Reference.Equal(run) {
		t.Fatalf("field = %+v", field.Grid)
	}
	if math.Abs(float64(field.Values[5]-values[5])) > 0.051 {
		t.Fatalf("value = %v, want %v", field.Values[5], values[5])
	}
	cropped, _, err := Crop(field.Grid, field.Values, Region)
	if err != nil || cropped.Lon0 != 70 || cropped.Lat0 != 18 || cropped.Nx != 23 || cropped.Ny != 21 {
		t.Fatalf("cropped = %+v %v", cropped, err)
	}
}
