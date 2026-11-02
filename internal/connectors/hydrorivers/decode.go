// Package hydrorivers loads Sri Lanka's river network: HydroRIVERS line
// segments with their flow topology, as extracted by nuuuwan/lk_rivers, and
// that repository's river names.
package hydrorivers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/prabhavalabs/lanka-weather/internal/geo"
)

// SourceName identifies the source in stored records and docs/data-sources.md.
const SourceName = "hydrosheds-hydrorivers"

// Attribution is the credit HydroSHEDS' licence asks for.
const Attribution = "HydroRIVERS (Lehner & Grill 2013, HydroSHEDS), extracted by nuuuwan/lk_rivers"

// Segment is one HydroRIVERS line, ordered from upstream to downstream.
type Segment struct {
	ID, NextDown, MainRiv int64
	LengthKM              float64
	DistDownKM, DistUpKM  float64
	CatchKM2, UplandKM2   float64
	DischargeCMS          float64
	OrdStra, OrdClas      int
	OrdFlow               int
	HybasL12              int64
	Line                  []geo.Point
}

type wireProperties struct {
	HyrivID   int64   `json:"HYRIV_ID"`
	NextDown  int64   `json:"NEXT_DOWN"`
	MainRiv   int64   `json:"MAIN_RIV"`
	LengthKM  float64 `json:"LENGTH_KM"`
	DistDown  float64 `json:"DIST_DN_KM"`
	DistUp    float64 `json:"DIST_UP_KM"`
	Catch     float64 `json:"CATCH_SKM"`
	Upland    float64 `json:"UPLAND_SKM"`
	Discharge float64 `json:"DIS_AV_CMS"`
	OrdStra   int     `json:"ORD_STRA"`
	OrdClas   int     `json:"ORD_CLAS"`
	OrdFlow   int     `json:"ORD_FLOW"`
	// The source writes the sub-basin id as a float: 4120050170.0.
	HybasL12 float64 `json:"HYBAS_L12"`
}

// DecodeSegments reads the GeoJSON of segments. Features are MultiLineStrings
// of one line in the published file; a LineString is accepted too, and the
// parts of a longer MultiLineString are joined in order. Segments without an
// id or a line are skipped and counted. It does no I/O.
func DecodeSegments(payload []byte) (segments []Segment, skipped int, err error) {
	var collection struct {
		Features []struct {
			Properties wireProperties `json:"properties"`
			Geometry   struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(payload, &collection); err != nil {
		return nil, 0, fmt.Errorf("decode river segments: %w", err)
	}
	if collection.Features == nil {
		return nil, 0, errors.New("decode river segments: no features")
	}
	for _, feature := range collection.Features {
		line, err := decodeLine(feature.Geometry.Type, feature.Geometry.Coordinates)
		if err != nil || len(line) < 2 || feature.Properties.HyrivID == 0 {
			skipped++
			continue
		}
		p := feature.Properties
		segments = append(segments, Segment{
			ID: p.HyrivID, NextDown: p.NextDown, MainRiv: p.MainRiv, LengthKM: p.LengthKM,
			DistDownKM: p.DistDown, DistUpKM: p.DistUp, CatchKM2: p.Catch, UplandKM2: p.Upland,
			DischargeCMS: p.Discharge, OrdStra: p.OrdStra, OrdClas: p.OrdClas, OrdFlow: p.OrdFlow,
			HybasL12: int64(p.HybasL12), Line: line,
		})
	}
	return segments, skipped, nil
}

func decodeLine(kind string, coordinates json.RawMessage) ([]geo.Point, error) {
	switch kind {
	case "LineString":
		var line []geo.Point
		err := json.Unmarshal(coordinates, &line)
		return line, err
	case "MultiLineString":
		var parts [][]geo.Point
		if err := json.Unmarshal(coordinates, &parts); err != nil {
			return nil, err
		}
		var line []geo.Point
		for _, part := range parts {
			if len(line) > 0 && len(part) > 0 && line[len(line)-1] == part[0] {
				part = part[1:]
			}
			line = append(line, part...)
		}
		return line, nil
	}
	return nil, fmt.Errorf("unexpected geometry %q", kind)
}

// DecodeNames reads the id-to-name table. Blank names and "(Unnamed)" say
// nothing and are dropped, as are ids that are not numbers.
func DecodeNames(payload []byte) (map[int64]string, error) {
	var raw map[string]string
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("decode river names: %w", err)
	}
	names := make(map[int64]string, len(raw))
	for key, name := range raw {
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" || strings.EqualFold(name, "(unnamed)") {
			continue
		}
		names[id] = name
	}
	return names, nil
}

func lineWKB(segment Segment) []byte { return geo.LineWKB(segment.Line) }
