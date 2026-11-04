// Package imerg collects NASA's IMERG half-hourly precipitation imagery over
// Sri Lanka from the GIBS tile service, so browsers read the images from this
// API and never from NASA (ADR 0012).
//
// IMERG is a satellite estimate, not a measurement, and is published about
// six hours after the half hour it covers. GIBS renders it at 0.1° on the
// GoogleMapsCompatible_Level6 tile matrix set, so zoom 6 is the deepest zoom
// there is. The map's lowest zoom (5) already draws zoom 6 tiles at 256
// pixels, which makes zoom 6 the only one needed.
package imerg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/png"
	"strings"
	"time"
)

// SourceName identifies the source in stored records and docs/data-sources.md.
const SourceName = "nasa-gibs-imerg"

// Attribution is the credit NASA asks for.
const Attribution = "NASA GIBS, GPM IMERG"

// Layer and matrix set at GIBS. The 30-minute layer lists the half hours it
// has; IMERG_Precipitation_Rate holds only one image a day.
const (
	Layer         = "IMERG_Precipitation_Rate_30min"
	TileMatrixSet = "GoogleMapsCompatible_Level6"
	baseURL       = "https://gibs.earthdata.nasa.gov/wmts/epsg3857/best"
)

// Step is the time between images; an image covers the half hour from its
// time.
const Step = 30 * time.Minute

// Zoom is the only zoom collected, the deepest GIBS renders.
const Zoom = 6

// Tile is one tile of the collected set.
type Tile struct{ Z, X, Y int }

// Tile rows and columns of the collected square: 73.125 to 90° E and 0 to
// 16.64° N, which is Sri Lanka in the middle tile and the rain approaching
// over the seas around it.
const (
	firstColumn = 45
	lastColumn  = 47
	firstRow    = 29
	lastRow     = 31
)

// Bounds is the area the tiles cover: west, south, east, north.
var Bounds = [4]float64{73.125, 0, 90, 16.636192}

// Tiles lists the collected tiles.
func Tiles() []Tile {
	tiles := make([]Tile, 0, (lastColumn-firstColumn+1)*(lastRow-firstRow+1))
	for y := firstRow; y <= lastRow; y++ {
		for x := firstColumn; x <= lastColumn; x++ {
			tiles = append(tiles, Tile{Z: Zoom, X: x, Y: y})
		}
	}
	return tiles
}

// Covers reports whether a tile is in the collected set.
func Covers(tile Tile) bool {
	return tile.Z == Zoom && tile.X >= firstColumn && tile.X <= lastColumn && tile.Y >= firstRow && tile.Y <= lastRow
}

// TimeID is a time as it is written in our tile URLs; it needs no escaping.
func TimeID(at time.Time) string { return at.UTC().Format("20060102T1504Z") }

// ParseTimeID reads a time written by TimeID.
func ParseTimeID(id string) (time.Time, error) {
	return time.Parse("20060102T1504Z", id)
}

// TileURL is a tile's address at GIBS.
func TileURL(at time.Time, tile Tile) string {
	return fmt.Sprintf("%s/%s/default/%s/%s/%d/%d/%d.png", baseURL, Layer, at.UTC().Format(time.RFC3339), TileMatrixSet, tile.Z, tile.Y, tile.X)
}

// DomainsURL is the WMTS "domains" request listing the half hours in a time
// range. It answers in a few hundred bytes, against 5.8 MB for the service's
// capabilities document.
func DomainsURL(from, to time.Time) string {
	return fmt.Sprintf("%s/1.0.0/%s/default/%s/all/%s--%s.xml", baseURL, Layer, TileMatrixSet,
		from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339))
}

// Interval is a run of images, Period apart, from Start to End inclusive.
type Interval struct {
	Start, End time.Time
	Period     time.Duration
}

type wireDomains struct {
	Dimensions []struct {
		Identifier string `xml:"Identifier"`
		Domain     string `xml:"Domain"`
	} `xml:"DimensionDomain"`
}

// DecodeDomains reads a WMTS domains response. A time domain is one or more
// comma-separated `start/end/period` ranges. It does no I/O.
func DecodeDomains(payload []byte) ([]Interval, error) {
	var wire wireDomains
	if err := xml.Unmarshal(payload, &wire); err != nil {
		return nil, fmt.Errorf("decode domains: %w", err)
	}
	for _, dimension := range wire.Dimensions {
		if !strings.EqualFold(dimension.Identifier, "time") {
			continue
		}
		var intervals []Interval
		for part := range strings.SplitSeq(dimension.Domain, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			fields := strings.Split(part, "/")
			if len(fields) != 3 {
				return nil, fmt.Errorf("time domain %q is not start/end/period", part)
			}
			start, err := parseDomainTime(fields[0])
			if err != nil {
				return nil, err
			}
			end, err := parseDomainTime(fields[1])
			if err != nil {
				return nil, err
			}
			period, err := parseMinutes(fields[2])
			if err != nil {
				return nil, err
			}
			intervals = append(intervals, Interval{Start: start, End: end, Period: period})
		}
		return intervals, nil
	}
	return nil, errors.New("domains response has no time dimension")
}

// parseDomainTime reads a domain's time: GIBS writes the start of a range as
// a bare date when it falls on one.
func parseDomainTime(text string) (time.Time, error) {
	if at, err := time.Parse(time.RFC3339, text); err == nil {
		return at.UTC(), nil
	}
	if at, err := time.Parse(time.DateOnly, text); err == nil {
		return at.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("time domain has unreadable time %q", text)
}

func parseMinutes(text string) (time.Duration, error) {
	var minutes int
	if _, err := fmt.Sscanf(text, "PT%dM", &minutes); err != nil || minutes <= 0 {
		return 0, fmt.Errorf("time domain has unreadable period %q", text)
	}
	return time.Duration(minutes) * time.Minute, nil
}

// LatestSteps returns up to `count` of the newest image times in the
// intervals at or after `from`, newest first.
func LatestSteps(intervals []Interval, from time.Time, count int) []time.Time {
	seen := map[time.Time]bool{}
	var steps []time.Time
	for _, interval := range intervals {
		// Steps count back from the end: a date at the start of a range need
		// not fall on a step.
		for at := interval.End; !at.Before(interval.Start) && !at.Before(from); at = at.Add(-interval.Period) {
			if !seen[at] {
				seen[at] = true
				steps = append(steps, at)
			}
		}
	}
	for i := 1; i < len(steps); i++ {
		for j := i; j > 0 && steps[j].After(steps[j-1]); j-- {
			steps[j], steps[j-1] = steps[j-1], steps[j]
		}
	}
	if len(steps) > count {
		steps = steps[:count]
	}
	return steps
}

// Inspect checks that a body is a PNG and reports whether every pixel of it
// is transparent. A tile with no rain is such an image, and is not a failure.
func Inspect(body []byte) (empty bool, err error) {
	decoded, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("decode tile: %w", err)
	}
	return transparent(decoded), nil
}

func transparent(picture image.Image) bool {
	bounds := picture.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if _, _, _, alpha := picture.At(x, y).RGBA(); alpha != 0 {
				return false
			}
		}
	}
	return true
}
