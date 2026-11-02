// Package osmreservoirs gives reservoirs a place on the map. The reservoir feed
// has no coordinates, so named water bodies and dams are read from OpenStreetMap
// through the Overpass API and matched to reservoirs by normalised name or a
// sourced alias, inside the reservoir's district.
//
// Matching is deliberately conservative: Sri Lanka has many small tanks with the
// same name, and a reservoir drawn in the wrong place is worse than one that is
// not drawn. A reservoir that does not match stays off the map and its panel
// says "Location not mapped".
//
// OpenStreetMap data is © OpenStreetMap contributors and available under the
// Open Database Licence (ODbL).
package osmreservoirs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SourceName identifies this source in docs/data-sources.md and source_runs.
const SourceName = "osm-reservoirs"

// Attribution is the credit OpenStreetMap's licence requires.
const Attribution = "© OpenStreetMap contributors (ODbL)"

// The kinds of candidate.
const (
	KindWater = "water"
	KindDam   = "dam"
)

// Candidate is a named water body or dam from OpenStreetMap.
type Candidate struct {
	// OSMID is "way/123", "relation/123" or "node/123".
	OSMID string
	Name  string
	// Kind is water or dam.
	Kind string
	// Names are all the names the feature is tagged with.
	Names []string
	// Lines is the feature's geometry as a GeoJSON line, multi-line or point;
	// the database builds the polygon, which needs the rings of a relation
	// stitched together.
	Lines string
}

// Snapshot is one decoded Overpass response.
type Snapshot struct {
	Candidates []Candidate
	// Elements counts every element in the response, used or not.
	Elements int
	// SourceTime is when OpenStreetMap's database was last updated before the
	// query ran.
	SourceTime time.Time
}

type latLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type response struct {
	OSM3S struct {
		TimestampOSMBase string `json:"timestamp_osm_base"`
	} `json:"osm3s"`
	// Remark reports a query that stopped early; its elements are incomplete.
	Remark   string `json:"remark"`
	Elements []struct {
		Type     string            `json:"type"`
		ID       int64             `json:"id"`
		Lat      *float64          `json:"lat"`
		Lon      *float64          `json:"lon"`
		Tags     map[string]string `json:"tags"`
		Geometry []latLon          `json:"geometry"`
		Members  []struct {
			Type     string   `json:"type"`
			Role     string   `json:"role"`
			Geometry []latLon `json:"geometry"`
		} `json:"members"`
	} `json:"elements"`
}

// nonReservoirWater are values of the water tag that are never a reservoir.
var nonReservoirWater = map[string]bool{
	"river": true, "stream": true, "canal": true, "lagoon": true, "oxbow": true,
	"pool": true, "wastewater": true, "stream_pool": true, "moat": true,
}

// Decode turns an Overpass JSON response into candidates. It does no I/O.
func Decode(payload []byte) (Snapshot, error) {
	var body response
	if err := json.Unmarshal(payload, &body); err != nil {
		return Snapshot{}, fmt.Errorf("decode overpass response: %w", err)
	}
	if body.Remark != "" {
		return Snapshot{}, fmt.Errorf("overpass query incomplete: %s", body.Remark)
	}
	sourceTime, err := time.Parse(time.RFC3339, body.OSM3S.TimestampOSMBase)
	if err != nil {
		return Snapshot{}, errors.New("overpass response has no database timestamp")
	}
	snapshot := Snapshot{SourceTime: sourceTime.UTC(), Elements: len(body.Elements)}
	for _, element := range body.Elements {
		kind := candidateKind(element.Tags)
		if kind == "" {
			continue
		}
		names := tagNames(element.Tags)
		if len(names) == 0 {
			continue
		}
		var coordinates [][][2]float64
		switch element.Type {
		case "way":
			coordinates = [][][2]float64{line(element.Geometry)}
		case "relation":
			for _, member := range element.Members {
				if member.Type == "way" && len(member.Geometry) >= 2 {
					coordinates = append(coordinates, line(member.Geometry))
				}
			}
		}
		lines, ok := geoJSON(element.Type, element.Lon, element.Lat, coordinates)
		if !ok {
			continue
		}
		snapshot.Candidates = append(snapshot.Candidates, Candidate{
			OSMID: element.Type + "/" + strconv.FormatInt(element.ID, 10),
			Name:  names[0], Kind: kind, Names: names, Lines: lines,
		})
	}
	return snapshot, nil
}

func candidateKind(tags map[string]string) string {
	if tags["man_made"] == "dam" || tags["waterway"] == "dam" {
		return KindDam
	}
	if tags["landuse"] == "reservoir" {
		return KindWater
	}
	if tags["natural"] == "water" && !nonReservoirWater[strings.ToLower(tags["water"])] {
		return KindWater
	}
	return ""
}

// tagNames returns the feature's names, its English one first. Several names in
// one tag are separated by ";" and, in Sri Lankan tanks, often by "/".
func tagNames(tags map[string]string) []string {
	var out []string
	seen := map[string]bool{}
	for _, key := range []string{"name:en", "name", "alt_name", "alt_name:en", "official_name", "old_name", "loc_name", "short_name"} {
		for _, part := range strings.FieldsFunc(tags[key], func(r rune) bool { return r == ';' || r == '/' }) {
			name := strings.Join(strings.Fields(part), " ")
			if name != "" && !seen[strings.ToLower(name)] {
				seen[strings.ToLower(name)] = true
				out = append(out, name)
			}
		}
	}
	return out
}

func line(points []latLon) [][2]float64 {
	out := make([][2]float64, len(points))
	for index, point := range points {
		out[index] = [2]float64{point.Lon, point.Lat}
	}
	return out
}

func geoJSON(elementType string, lon, lat *float64, coordinates [][][2]float64) (string, bool) {
	var value any
	switch {
	case elementType == "node":
		if lon == nil || lat == nil {
			return "", false
		}
		value = map[string]any{"type": "Point", "coordinates": [2]float64{*lon, *lat}}
	case len(coordinates) == 0 || len(coordinates[0]) < 2:
		return "", false
	case elementType == "way":
		value = map[string]any{"type": "LineString", "coordinates": coordinates[0]}
	default:
		value = map[string]any{"type": "MultiLineString", "coordinates": coordinates}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	return string(encoded), true
}
