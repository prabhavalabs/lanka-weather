// Package osmplaces collects Sri Lanka's populated places (cities, towns,
// villages, hamlets, suburbs and named localities) from OpenStreetMap
// through the Overpass API, for place search.
//
// OpenStreetMap data is © OpenStreetMap contributors and available under
// the Open Database Licence (ODbL).
package osmplaces

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/names"
)

// SourceName identifies this source in docs/data-sources.md and source_runs.
const SourceName = "osm-places"

// Attribution is the credit OpenStreetMap's licence requires.
const Attribution = "© OpenStreetMap contributors (ODbL)"

// Kinds are the OpenStreetMap place values collected, from most to least
// important.
var Kinds = []string{"city", "town", "suburb", "village", "quarter", "neighbourhood", "hamlet", "locality", "isolated_dwelling"}

// Place is one populated place.
type Place struct {
	// ID is "osm-node-" and the node's id.
	ID     string
	Name   string
	NameSI string
	NameTA string
	// AltNames are other names the place is known by, for search.
	AltNames   []string
	Kind       string
	Population *int32
	Lon, Lat   float64
}

// Snapshot is one decoded Overpass response.
type Snapshot struct {
	Places []Place
	// SourceTime is when OpenStreetMap's database was last updated before the
	// query ran (osm3s.timestamp_osm_base).
	SourceTime time.Time
	// Skipped counts elements that could not be used.
	Skipped int
}

type response struct {
	OSM3S struct {
		TimestampOSMBase string `json:"timestamp_osm_base"`
	} `json:"osm3s"`
	// Remark reports a query that stopped early, for example on a timeout;
	// its elements are then incomplete.
	Remark   string `json:"remark"`
	Elements []struct {
		Type string            `json:"type"`
		ID   int64             `json:"id"`
		Lat  *float64          `json:"lat"`
		Lon  *float64          `json:"lon"`
		Tags map[string]string `json:"tags"`
	} `json:"elements"`
}

// The query's box around Sri Lanka. It reaches no part of India, whose
// nearest coast is west of 79.5°E.
const minLon, maxLon, minLat, maxLat = 79.5, 81.95, 5.85, 9.9

// maxAltNames bounds the alternative names kept per place.
const maxAltNames = 8

// Decode turns an Overpass JSON response into places. It does no I/O.
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
	kinds := make(map[string]bool, len(Kinds))
	for _, kind := range Kinds {
		kinds[kind] = true
	}
	snapshot := Snapshot{SourceTime: sourceTime.UTC()}
	for _, element := range body.Elements {
		tags := element.Tags
		kind := strings.TrimSpace(tags["place"])
		if element.Type != "node" || element.Lat == nil || element.Lon == nil || !kinds[kind] ||
			!inSriLanka(*element.Lon, *element.Lat) {
			snapshot.Skipped++
			continue
		}
		place, ok := placeNames(tags)
		if !ok {
			snapshot.Skipped++
			continue
		}
		place.ID = "osm-node-" + strconv.FormatInt(element.ID, 10)
		place.Kind = kind
		place.Lon, place.Lat = *element.Lon, *element.Lat
		place.Population = ParsePopulation(tags["population"])
		snapshot.Places = append(snapshot.Places, place)
	}
	return snapshot, nil
}

// placeNames chooses the display name and the Sinhala, Tamil and other
// names. The display name is the English name when there is one, otherwise
// the name tag, which in Sri Lanka is usually in Latin script but sometimes
// in Sinhala or Tamil.
func placeNames(tags map[string]string) (Place, bool) {
	base := clean(tags["name"])
	english := clean(tags["name:en"])
	var place Place
	switch {
	case english != "":
		place.Name = english
	case base != "":
		place.Name = base
	default:
		return Place{}, false
	}
	place.NameSI = clean(tags["name:si"])
	place.NameTA = clean(tags["name:ta"])
	switch names.Of(base) {
	case names.ScriptSinhala:
		if place.NameSI == "" {
			place.NameSI = base
		}
	case names.ScriptTamil:
		if place.NameTA == "" {
			place.NameTA = base
		}
	}
	seen := map[string]bool{strings.ToLower(place.Name): true}
	for _, known := range []string{place.NameSI, place.NameTA} {
		if known != "" {
			seen[strings.ToLower(known)] = true
		}
	}
	for _, key := range []string{"name", "alt_name", "old_name", "official_name", "loc_name", "short_name", "alt_name:en", "old_name:en"} {
		for _, name := range strings.Split(tags[key], ";") {
			name = clean(name)
			if name == "" || seen[strings.ToLower(name)] || len(place.AltNames) == maxAltNames {
				continue
			}
			seen[strings.ToLower(name)] = true
			place.AltNames = append(place.AltNames, name)
		}
	}
	return place, true
}

func clean(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

// ParsePopulation reads OpenStreetMap's free-text population tag, accepting
// thousands separators and an approximation sign; ranges and words are
// rejected.
func ParsePopulation(raw string) *int32 {
	value := strings.TrimSpace(raw)
	value = strings.TrimLeft(value, "~≈")
	value = strings.NewReplacer(",", "", " ", "", "_", "").Replace(value)
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil || number <= 0 || number > math.MaxInt32 {
		return nil
	}
	population := int32(number)
	return &population
}

func inSriLanka(lon, lat float64) bool {
	return lon >= minLon && lon <= maxLon && lat >= minLat && lat <= maxLat
}
