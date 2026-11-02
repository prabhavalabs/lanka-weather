package imerg

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"
)

func TestDecodeRecordedDomains(t *testing.T) {
	payload, err := os.ReadFile("testdata/domains.xml")
	if err != nil {
		t.Fatal(err)
	}
	intervals, err := DecodeDomains(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(intervals) != 1 || intervals[0].Period != 30*time.Minute {
		t.Fatalf("intervals = %+v", intervals)
	}
	if want := time.Date(2026, 11, 2, 13, 30, 0, 0, time.UTC); !intervals[0].End.Equal(want) {
		t.Fatalf("end = %s, want %s", intervals[0].End, want)
	}
	if !intervals[0].Start.Equal(time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("a bare date should start at midnight, got %s", intervals[0].Start)
	}
}

func TestDecodeDomainsRejectsWhatItCannotRead(t *testing.T) {
	for name, payload := range map[string]string{
		"no time":    `<Domains><DimensionDomain><Identifier>x</Identifier></DimensionDomain></Domains>`,
		"bad range":  `<Domains><DimensionDomain><Identifier>time</Identifier><Domain>2026-11-02</Domain></DimensionDomain></Domains>`,
		"bad period": `<Domains><DimensionDomain><Identifier>time</Identifier><Domain>2026-11-02/2026-11-02/P1D</Domain></DimensionDomain></Domains>`,
		"not xml":    `oops`,
		"bad end":    `<Domains><DimensionDomain><Identifier>time</Identifier><Domain>2026-11-02/soon/PT30M</Domain></DimensionDomain></Domains>`,
	} {
		if _, err := DecodeDomains([]byte(payload)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestLatestStepsCountsBackFromTheEndAndSkipsGaps(t *testing.T) {
	at := func(hour, minute int) time.Time { return time.Date(2026, 11, 2, hour, minute, 0, 0, time.UTC) }
	intervals := []Interval{
		{Start: at(0, 0), End: at(1, 0), Period: Step},
		{Start: at(3, 0), End: at(4, 0), Period: Step},
	}
	got := LatestSteps(intervals, at(0, 30), 4)
	want := []time.Time{at(4, 0), at(3, 30), at(3, 0), at(1, 0)}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for index := range want {
		if !got[index].Equal(want[index]) {
			t.Fatalf("step %d = %s, want %s", index, got[index], want[index])
		}
	}
}

func TestTilesCoverSriLankaInTheMiddle(t *testing.T) {
	tiles := Tiles()
	if len(tiles) != 9 {
		t.Fatalf("tiles = %d, want 9", len(tiles))
	}
	if !Covers(Tile{Z: 6, X: 46, Y: 30}) || Covers(Tile{Z: 6, X: 44, Y: 30}) || Covers(Tile{Z: 5, X: 46, Y: 30}) {
		t.Fatal("Covers disagrees with the collected square")
	}
	want := "https://gibs.earthdata.nasa.gov/wmts/epsg3857/best/IMERG_Precipitation_Rate_30min/default/2026-11-02T13:30:00Z/GoogleMapsCompatible_Level6/6/30/46.png"
	if got := TileURL(time.Date(2026, 11, 2, 13, 30, 0, 0, time.UTC), Tile{Z: 6, X: 46, Y: 30}); got != want {
		t.Fatalf("tile URL = %s", got)
	}
}

func TestTimeIDRoundTrips(t *testing.T) {
	at := time.Date(2026, 11, 2, 13, 30, 0, 0, time.UTC)
	if TimeID(at) != "20261102T1330Z" {
		t.Fatalf("id = %s", TimeID(at))
	}
	back, err := ParseTimeID(TimeID(at))
	if err != nil || !back.Equal(at) {
		t.Fatalf("back = %s, %v", back, err)
	}
	if _, err := ParseTimeID("../../etc"); err == nil {
		t.Fatal("expected an error for a malformed id")
	}
}

// paletted builds a tile as GIBS does: a palette whose first entry is
// transparent. Real tiles are not committed.
func paletted(t *testing.T, rain bool) []byte {
	t.Helper()
	palette := color.Palette{color.NRGBA{191, 191, 191, 0}, color.NRGBA{0, 165, 3, 255}}
	picture := image.NewPaletted(image.Rect(0, 0, 256, 256), palette)
	if rain {
		picture.SetColorIndex(200, 40, 1)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, picture); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestInspectFindsEmptyTilesWithoutFailing(t *testing.T) {
	if empty, err := Inspect(paletted(t, false)); err != nil || !empty {
		t.Fatalf("all transparent: empty = %v, err = %v", empty, err)
	}
	if empty, err := Inspect(paletted(t, true)); err != nil || empty {
		t.Fatalf("one rainy pixel: empty = %v, err = %v", empty, err)
	}
	if _, err := Inspect([]byte("<ServiceException/>")); err == nil {
		t.Fatal("a body that is not a PNG is a failure")
	}
}

func TestLegendStopsAreWhatGIBSRenders(t *testing.T) {
	payload, err := os.ReadFile("testdata/colormap_excerpt.xml")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := DecodeColormap(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, stop := range LegendStops {
		found := false
		for _, entry := range entries {
			if stop.RateMmH >= entry.From && stop.RateMmH < entry.To {
				found = true
				if entry.Color != stop.Color {
					t.Errorf("%v mm/h is drawn %s, legend says %s", stop.RateMmH, entry.Color, stop.Color)
				}
			}
		}
		if !found {
			t.Errorf("no colour-map class holds %v mm/h", stop.RateMmH)
		}
	}
}
