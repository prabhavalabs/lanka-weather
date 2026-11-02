// Package hmisrain collects live rain gauges from HMIS, the Hydro-Meteorological
// Information System of the Irrigation Department (hmis.gov.lk), through the copy
// Sri Lanka Water Levels publishes. HMIS's own endpoints sit behind its login in
// the site itself, so they are not used, however they answer (ADR 0015). None of
// that project's code is used: its repository has no licence.
//
// Each device gives points of [epoch minute, rolling 24-hour sum in mm]. A rolling
// sum cannot be turned into shorter totals, so only the 24-hour total and its time
// are ever shown.
package hmisrain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	// SourceName identifies the source in docs/data-sources.md and source_runs.
	SourceName = "hmis-rain-live"
	// Endpoint is the copy the owner approved, refreshed about every 30 minutes.
	Endpoint = "https://raw.githubusercontent.com/shakee93/sl-water-levels/data/hmis/rain-live.json"
	// Attribution is the credit for anything read from it; the operating
	// institution is credited beside it on each gauge.
	Attribution = "HMIS, via Sri Lanka Water Levels (waterlevels.lk)"
	// NotStated stands in for a device with no operating institution.
	NotStated = "Not stated"
	// highestPlausibleMm rejects a reading no 24 hours could produce: Sri Lanka's
	// record for a day is under 500 mm.
	highestPlausibleMm = 1500.0
)

// Sri Lanka with a margin; a gauge outside it is dropped.
const minLon, maxLon, minLat, maxLat = 79.0, 82.5, 5.5, 10.5

// Point is one reading: a rolling 24-hour sum ending at At.
type Point struct {
	At     time.Time
	RainMm float64
}

// Gauge is one device with its points, oldest first.
type Gauge struct {
	ID          string
	Device      string
	Name        string
	Institution string
	Lon, Lat    float64
	Points      []Point
}

// Newest is the gauge's latest point, if it has one.
func (gauge Gauge) Newest() (Point, bool) {
	if len(gauge.Points) == 0 {
		return Point{}, false
	}
	return gauge.Points[len(gauge.Points)-1], true
}

// Feed is one decoded file.
type Feed struct {
	UpdatedAt time.Time
	Gauges    []Gauge
	// Skipped counts devices that could not be used: no usable position or name.
	Skipped int
}

type wireDevice struct {
	Name        string       `json:"n"`
	Institution string       `json:"i"`
	Lat         *float64     `json:"lat"`
	Lon         *float64     `json:"lon"`
	Points      [][2]float64 `json:"pts"`
}

// Decode reads the file. A device with no usable points is kept, since it says the
// gauge exists and has not reported; it does no I/O.
func Decode(payload []byte) (Feed, error) {
	var body struct {
		UpdatedAt string                `json:"updatedAt"`
		Devices   map[string]wireDevice `json:"devices"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return Feed{}, fmt.Errorf("decode rain gauges: %w", err)
	}
	updated, err := time.Parse(time.RFC3339, body.UpdatedAt)
	if err != nil {
		return Feed{}, errors.New("rain gauges file has no updatedAt")
	}
	feed := Feed{UpdatedAt: updated.UTC()}
	seen := make(map[string]bool, len(body.Devices))
	for device, wire := range body.Devices {
		id := Slug(device)
		name := Tidy(wire.Name)
		if id == "" || name == "" || wire.Lat == nil || wire.Lon == nil || seen[id] ||
			*wire.Lon < minLon || *wire.Lon > maxLon || *wire.Lat < minLat || *wire.Lat > maxLat {
			feed.Skipped++
			continue
		}
		seen[id] = true
		institution := strings.TrimSpace(wire.Institution)
		if institution == "" {
			institution = NotStated
		}
		feed.Gauges = append(feed.Gauges, Gauge{ID: id, Device: strings.TrimSpace(device), Name: name, Institution: institution,
			Lon: *wire.Lon, Lat: *wire.Lat, Points: points(wire.Points, feed.UpdatedAt)})
	}
	sort.Slice(feed.Gauges, func(i, j int) bool { return feed.Gauges[i].ID < feed.Gauges[j].ID })
	return feed, nil
}

// points keeps plausible readings not later than the file itself, oldest first.
// Minutes are real UTC: on 2 November 2026 the median newest reading was 18 minutes
// before the file's own time.
func points(wire [][2]float64, updated time.Time) []Point {
	out := make([]Point, 0, len(wire))
	for _, point := range wire {
		rain := point[1]
		if math.IsNaN(rain) || rain < 0 || rain > highestPlausibleMm || point[0] <= 0 {
			continue
		}
		at := time.Unix(int64(point[0])*60, 0).UTC()
		if at.After(updated.Add(time.Hour)) {
			continue
		}
		out = append(out, Point{At: at, RainMm: rain})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

var (
	notSlug  = regexp.MustCompile(`[^a-z0-9]+`)
	spaces   = regexp.MustCompile(`\s+`)
	badComma = regexp.MustCompile(`\s+,\s*`)
)

// Slug makes a device's name safe in a URL.
func Slug(device string) string {
	return strings.Trim(notSlug.ReplaceAllString(strings.ToLower(device), "-"), "-")
}

// Tidy cleans how a name is written and changes no words: "ASC_Damana" reads "ASC
// Damana" and "Center ,Alapilli" reads "Center, Alapilli".
func Tidy(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	name = badComma.ReplaceAllString(name, ", ")
	return strings.TrimSpace(spaces.ReplaceAllString(name, " "))
}
