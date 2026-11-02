// Package googleflood collects Google's Flood Forecasting API (the data
// behind Google Flood Hub) for Sri Lanka only.
//
// It reads the latest flood status of every gauge in Sri Lanka (searched by
// region code "LK"), the gauges' warning thresholds, and the flash floods
// and significant flood events whose affected countries include Sri Lanka,
// with the polygons Google publishes for anything above normal. Every
// request carries the API key in the X-Goog-Api-Key header, never in a URL,
// so it cannot reach logs. Google's terms allow gauge search results to be
// kept for at most a day.
//
// Adapted from TerraGuard's connector (ADR 0007).
package googleflood

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// SourceName identifies the source in stored records and docs/data-sources.md.
const SourceName = "google-flood-forecasting"

// Attribution is the credit shown with the layer.
const Attribution = "Flood forecasts: Google Flood Hub (Flood Forecasting API)"

// RegionCode is the only region queried.
const RegionCode = "LK"

// Severity is Google's forecast flood severity at a gauge, lower-cased.
type Severity string

const (
	SeverityNoFlooding  Severity = "no_flooding"
	SeverityAboveNormal Severity = "above_normal"
	SeveritySevere      Severity = "severe"
	SeverityExtreme     Severity = "extreme"
	SeverityUnknown     Severity = "unknown"
)

// Rank orders severities from unknown (0) to extreme (4).
func (severity Severity) Rank() int {
	switch severity {
	case SeverityNoFlooding:
		return 1
	case SeverityAboveNormal:
		return 2
	case SeveritySevere:
		return 3
	case SeverityExtreme:
		return 4
	}
	return 0
}

// Status is the latest forecast at one gauge.
type Status struct {
	GaugeID         string
	GaugeSource     string
	SiteName        string
	River           string
	Lon, Lat        float64
	QualityVerified bool
	Severity        Severity
	// Trend is "rise", "fall", "no_change" or "unknown".
	Trend         string
	IssuedAt      time.Time
	ForecastStart *time.Time
	ForecastEnd   *time.Time
	// NotificationPolygon and InundationPolygons ("high", "medium", "low")
	// are polygon IDs, given for gauges above normal.
	NotificationPolygon string
	InundationPolygons  map[string]string
	// InundationType says what the levels mean: "probability" (high inside
	// medium inside low) or "depth", from Google's InundationMapType.
	InundationType string
	// InundationStart and InundationEnd are the state the maps refer to.
	InundationStart *time.Time
	InundationEnd   *time.Time
}

type wireInterval struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type wireStatus struct {
	Source        string `json:"source"`
	GaugeID       string `json:"gaugeId"`
	GaugeName     string `json:"gaugeName"`
	RiverName     string `json:"riverName"`
	Severity      string `json:"severity"`
	IssuedTime    string `json:"issuedTime"`
	ForecastTrend string `json:"forecastTrend"`
	GaugeLocation *struct {
		// Google omits zero coordinates.
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"gaugeLocation"`
	QualityVerified   bool          `json:"qualityVerified"`
	ForecastTimeRange *wireInterval `json:"forecastTimeRange"`
	NotificationID    string        `json:"serializedNotificationPolygonId"`
	InundationMapSet  *struct {
		InundationMaps []struct {
			Level     string `json:"level"`
			PolygonID string `json:"serializedPolygonId"`
		} `json:"inundationMaps"`
		TimeRange *wireInterval `json:"inundationMapsTimeRange"`
		MapType   string        `json:"inundationMapType"`
	} `json:"inundationMapSet"`
}

// Sri Lanka with a margin; gauges outside it are dropped.
const minLon, maxLon, minLat, maxLat = 79.0, 82.5, 5.5, 10.5

// DecodeStatuses reads one page of searchLatestFloodStatusByArea. Statuses
// that cannot be used (no gauge, location or issue time, or outside Sri
// Lanka) are counted in skipped. It does no I/O.
func DecodeStatuses(payload []byte) (statuses []Status, next string, skipped int, err error) {
	var page struct {
		FloodStatuses []wireStatus `json:"floodStatuses"`
		NextPageToken string       `json:"nextPageToken"`
	}
	if err := decodeObject(payload, &page); err != nil {
		return nil, "", 0, fmt.Errorf("decode flood statuses: %w", err)
	}
	for _, wire := range page.FloodStatuses {
		status, ok := decodeStatus(wire)
		if !ok {
			skipped++
			continue
		}
		statuses = append(statuses, status)
	}
	return statuses, strings.TrimSpace(page.NextPageToken), skipped, nil
}

func decodeStatus(wire wireStatus) (Status, bool) {
	id := strings.TrimSpace(wire.GaugeID)
	if id == "" || wire.GaugeLocation == nil {
		return Status{}, false
	}
	lon, lat := wire.GaugeLocation.Longitude, wire.GaugeLocation.Latitude
	if lon < minLon || lon > maxLon || lat < minLat || lat > maxLat {
		return Status{}, false
	}
	issued, err := parseTime(wire.IssuedTime)
	if err != nil || issued == nil {
		return Status{}, false
	}
	status := Status{
		GaugeID: id, GaugeSource: strings.TrimSpace(wire.Source), SiteName: strings.TrimSpace(wire.GaugeName),
		River: strings.TrimSpace(wire.RiverName), Lon: lon, Lat: lat, QualityVerified: wire.QualityVerified,
		Severity: severity(wire.Severity), Trend: trend(wire.ForecastTrend), IssuedAt: *issued,
		NotificationPolygon: strings.TrimSpace(wire.NotificationID),
	}
	if wire.ForecastTimeRange != nil {
		status.ForecastStart, _ = parseTime(wire.ForecastTimeRange.Start)
		status.ForecastEnd, _ = parseTime(wire.ForecastTimeRange.End)
	}
	if wire.InundationMapSet != nil {
		for _, inundation := range wire.InundationMapSet.InundationMaps {
			level := strings.ToLower(strings.TrimSpace(inundation.Level))
			id := strings.TrimSpace(inundation.PolygonID)
			if id != "" && (level == "high" || level == "medium" || level == "low") {
				if status.InundationPolygons == nil {
					status.InundationPolygons = map[string]string{}
				}
				status.InundationPolygons[level] = id
			}
		}
		status.InundationType = strings.ToLower(strings.TrimSpace(wire.InundationMapSet.MapType))
		if timeRange := wire.InundationMapSet.TimeRange; timeRange != nil {
			status.InundationStart, _ = parseTime(timeRange.Start)
			status.InundationEnd, _ = parseTime(timeRange.End)
		}
	}
	return status, true
}

func severity(value string) Severity {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "NO_FLOODING":
		return SeverityNoFlooding
	case "ABOVE_NORMAL":
		return SeverityAboveNormal
	case "SEVERE":
		return SeveritySevere
	case "EXTREME":
		return SeverityExtreme
	}
	return SeverityUnknown
}

func trend(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "RISE":
		return "rise"
	case "FALL":
		return "fall"
	case "NO_CHANGE":
		return "no_change"
	}
	return "unknown"
}

// Model is a gauge's forecast model: the warning, danger and extreme danger
// thresholds its forecasts are compared with, in Unit.
type Model struct {
	GaugeID                                 string
	WarningLevel, DangerLevel, ExtremeLevel *float64
	// Unit is "m3/s", "m" or "" when Google gives another unit.
	Unit string
}

// DecodeModels reads gaugeModels:batchGet. It does no I/O.
func DecodeModels(payload []byte) (map[string]Model, error) {
	var body struct {
		GaugeModels []struct {
			GaugeID    string `json:"gaugeId"`
			Thresholds *struct {
				WarningLevel       *float64 `json:"warningLevel"`
				DangerLevel        *float64 `json:"dangerLevel"`
				ExtremeDangerLevel *float64 `json:"extremeDangerLevel"`
			} `json:"thresholds"`
			GaugeValueUnit string `json:"gaugeValueUnit"`
		} `json:"gaugeModels"`
	}
	if err := decodeObject(payload, &body); err != nil {
		return nil, fmt.Errorf("decode gauge models: %w", err)
	}
	models := make(map[string]Model, len(body.GaugeModels))
	for _, wire := range body.GaugeModels {
		id := strings.TrimSpace(wire.GaugeID)
		if id == "" {
			continue
		}
		model := Model{GaugeID: id}
		switch wire.GaugeValueUnit {
		case "CUBIC_METERS_PER_SECOND":
			model.Unit = "m3/s"
		case "METERS":
			model.Unit = "m"
		}
		if wire.Thresholds != nil {
			model.WarningLevel = finite(wire.Thresholds.WarningLevel)
			model.DangerLevel = finite(wire.Thresholds.DangerLevel)
			model.ExtremeLevel = finite(wire.Thresholds.ExtremeDangerLevel)
		}
		models[id] = model
	}
	return models, nil
}

// Area is a flash flood or significant event affecting Sri Lanka, or the
// polygon of a gauge's forecast.
type Area struct {
	// PolygonID is Google's serialized polygon ID, fetched separately.
	PolygonID string
	Kind      string
	GaugeID   string
	// Severity is the gauge's severity, "likely" or "highly_likely" for a
	// flash flood, or the event's affected population band.
	Severity   string
	IssuedAt   *time.Time
	ValidFrom  *time.Time
	ValidUntil *time.Time
	Properties map[string]any
}

// Area kinds, as stored.
const (
	KindNotification     = "notification"
	KindFlashLikely      = "flash_likely"
	KindFlashHighly      = "flash_highly_likely"
	KindSignificantEvent = "significant_event"
)

// DecodeFlashFloods reads flashFloods:search and keeps the forecasts affecting
// Sri Lanka, one area per likely or highly likely polygon. eventPolygonId is
// not read: Google documents it as the union of the other two, and live it
// repeats the highly likely polygon, so drawing it would add nothing. It does
// no I/O.
func DecodeFlashFloods(payload []byte) ([]Area, string, error) {
	var page struct {
		FlashFloodEvents []struct {
			ForecastIssueTime     string   `json:"forecastIssueTime"`
			ForecastPeriodHours   number   `json:"forecastPeriodHours"`
			AffectedCountryCodes  []string `json:"affectedCountryCodes"`
			LikelyPolygonID       string   `json:"likelyAffectedPolygonId"`
			HighlyLikelyPolygonID string   `json:"highlyLikelyAffectedPolygonId"`
		} `json:"flashFloodEvents"`
		NextPageToken string `json:"nextPageToken"`
	}
	if err := decodeObject(payload, &page); err != nil {
		return nil, "", fmt.Errorf("decode flash floods: %w", err)
	}
	var areas []Area
	for _, event := range page.FlashFloodEvents {
		if !affectsSriLanka(event.AffectedCountryCodes) {
			continue
		}
		issued, err := parseTime(event.ForecastIssueTime)
		if err != nil || issued == nil {
			continue
		}
		var until *time.Time
		if hours := float64(event.ForecastPeriodHours); hours > 0 {
			end := issued.Add(time.Duration(hours * float64(time.Hour)))
			until = &end
		}
		for _, polygon := range []struct{ id, kind, severity string }{
			{event.HighlyLikelyPolygonID, KindFlashHighly, "highly_likely"},
			{event.LikelyPolygonID, KindFlashLikely, "likely"},
		} {
			if id := strings.TrimSpace(polygon.id); id != "" {
				areas = append(areas, Area{PolygonID: id, Kind: polygon.kind, Severity: polygon.severity,
					IssuedAt: issued, ValidFrom: issued, ValidUntil: until,
					Properties: map[string]any{"forecastPeriodHours": float64(event.ForecastPeriodHours)}})
			}
		}
	}
	return areas, strings.TrimSpace(page.NextPageToken), nil
}

// DecodeSignificantEvents reads significantEvents:search and keeps the
// events affecting Sri Lanka that have a polygon. It does no I/O.
func DecodeSignificantEvents(payload []byte) ([]Area, string, error) {
	var page struct {
		SignificantEvents []struct {
			AreaKm2       *number  `json:"areaKm2"`
			GaugeIDs      []string `json:"gaugeIds"`
			EventInterval struct {
				StartTime string `json:"startTime"`
				EndTime   string `json:"endTime"`
			} `json:"eventInterval"`
			EventPolygonID       string   `json:"eventPolygonId"`
			AffectedPopulation   *number  `json:"affectedPopulation"`
			AffectedCountryCodes []string `json:"affectedCountryCodes"`
		} `json:"significantEvents"`
		NextPageToken string `json:"nextPageToken"`
	}
	if err := decodeObject(payload, &page); err != nil {
		return nil, "", fmt.Errorf("decode significant events: %w", err)
	}
	var areas []Area
	for _, event := range page.SignificantEvents {
		id := strings.TrimSpace(event.EventPolygonID)
		if id == "" || !affectsSriLanka(event.AffectedCountryCodes) {
			continue
		}
		start, _ := parseTime(event.EventInterval.StartTime)
		end, _ := parseTime(event.EventInterval.EndTime)
		properties := map[string]any{"gaugeIds": event.GaugeIDs}
		band := "unknown"
		if event.AffectedPopulation != nil {
			people := float64(*event.AffectedPopulation)
			properties["affectedPopulation"] = people
			band = populationBand(people)
		}
		if event.AreaKm2 != nil {
			properties["areaKm2"] = float64(*event.AreaKm2)
		}
		areas = append(areas, Area{PolygonID: id, Kind: KindSignificantEvent, Severity: band,
			ValidFrom: start, ValidUntil: end, Properties: properties})
	}
	return areas, strings.TrimSpace(page.NextPageToken), nil
}

// populationBand grades a significant event by the people in its area, as
// TerraGuard does: 100,000 or more is severe, 50,000 or more moderate.
func populationBand(people float64) string {
	switch {
	case people >= 100_000:
		return "severe"
	case people >= 50_000:
		return "moderate"
	}
	return "minor"
}

// GaugeAreas lists the polygons of a gauge above normal: its notification
// polygon and its inundation maps.
func GaugeAreas(status Status) []Area {
	if status.Severity.Rank() < SeverityAboveNormal.Rank() {
		return nil
	}
	base := Area{GaugeID: status.GaugeID, Severity: string(status.Severity), IssuedAt: &status.IssuedAt,
		ValidFrom: status.ForecastStart, ValidUntil: status.ForecastEnd}
	var areas []Area
	if status.NotificationPolygon != "" {
		area := base
		area.PolygonID, area.Kind = status.NotificationPolygon, KindNotification
		areas = append(areas, area)
	}
	for _, level := range []string{"high", "medium", "low"} {
		if id := status.InundationPolygons[level]; id != "" {
			area := base
			area.PolygonID, area.Kind = id, "inundation_"+level
			if status.InundationStart != nil || status.InundationEnd != nil {
				area.ValidFrom, area.ValidUntil = status.InundationStart, status.InundationEnd
			}
			if status.InundationType != "" {
				area.Properties = map[string]any{"mapType": status.InundationType}
			}
			areas = append(areas, area)
		}
	}
	return areas
}

func affectsSriLanka(codes []string) bool {
	return slices.ContainsFunc(codes, func(code string) bool { return strings.EqualFold(strings.TrimSpace(code), RegionCode) })
}

// ParsePolygon reads serializedPolygons/{id}: Google's KML envelope, or a
// GeoJSON Polygon or MultiPolygon, and returns a GeoJSON MultiPolygon with
// closed rings. Degenerate rings are dropped; PostGIS repairs the rest.
func ParsePolygon(raw []byte) ([]byte, error) {
	var envelope struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
		Polygon     json.RawMessage `json:"polygon"`
		KML         string          `json:"kml"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("polygon JSON: %w", err)
	}
	var polygons [][][][2]float64
	switch {
	case envelope.Type == "Polygon":
		var polygon [][][2]float64
		if err := json.Unmarshal(envelope.Coordinates, &polygon); err != nil {
			return nil, err
		}
		polygons = [][][][2]float64{polygon}
	case envelope.Type == "MultiPolygon":
		if err := json.Unmarshal(envelope.Coordinates, &polygons); err != nil {
			return nil, err
		}
	case len(envelope.Polygon) > 0 && string(envelope.Polygon) != "null":
		return ParsePolygon(envelope.Polygon)
	case envelope.KML != "":
		var err error
		if polygons, err = parseKML(envelope.KML); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("polygon has no geometry or KML")
	}
	return cleanPolygons(polygons)
}

func parseKML(document string) ([][][][2]float64, error) {
	decoder := xml.NewDecoder(strings.NewReader(document))
	var polygons [][][][2]float64
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("polygon KML: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Polygon" {
			continue
		}
		var polygon struct {
			Outer string   `xml:"outerBoundaryIs>LinearRing>coordinates"`
			Inner []string `xml:"innerBoundaryIs>LinearRing>coordinates"`
		}
		if err := decoder.DecodeElement(&polygon, &start); err != nil {
			return nil, err
		}
		var rings [][][2]float64
		for _, coordinates := range append([]string{polygon.Outer}, polygon.Inner...) {
			var ring [][2]float64
			for _, tuple := range strings.Fields(coordinates) {
				fields := strings.Split(tuple, ",")
				if len(fields) < 2 {
					return nil, errors.New("polygon KML has an invalid coordinate")
				}
				lon, lonErr := strconv.ParseFloat(fields[0], 64)
				lat, latErr := strconv.ParseFloat(fields[1], 64)
				if lonErr != nil || latErr != nil {
					return nil, errors.New("polygon KML has an invalid number")
				}
				ring = append(ring, [2]float64{lon, lat})
			}
			rings = append(rings, ring)
		}
		polygons = append(polygons, rings)
	}
	return polygons, nil
}

// cleanPolygons closes rings, drops rings with fewer than three distinct
// points (and a polygon whose outer ring is one), and checks coordinates.
func cleanPolygons(polygons [][][][2]float64) ([]byte, error) {
	var cleaned [][][][2]float64
	vertices := 0
	for _, polygon := range polygons {
		var rings [][][2]float64
		for index, ring := range polygon {
			vertices += len(ring)
			if vertices > 200_000 {
				return nil, errors.New("polygon has more than 200,000 vertices")
			}
			distinct := map[[2]float64]bool{}
			for _, point := range ring {
				if !(point[0] >= -180 && point[0] <= 180 && point[1] >= -90 && point[1] <= 90) {
					return nil, errors.New("polygon coordinate outside valid bounds")
				}
				distinct[point] = true
			}
			if len(distinct) < 3 {
				if index == 0 {
					break
				}
				continue
			}
			if ring[0] != ring[len(ring)-1] {
				ring = append(ring, ring[0])
			}
			rings = append(rings, ring)
		}
		if len(rings) > 0 {
			cleaned = append(cleaned, rings)
		}
	}
	if len(cleaned) == 0 {
		return nil, errors.New("polygon has no usable ring")
	}
	return json.Marshal(struct {
		Type        string           `json:"type"`
		Coordinates [][][][2]float64 `json:"coordinates"`
	}{"MultiPolygon", cleaned})
}

// number reads a JSON number or a quoted one: Google's JSON mapping writes
// 64-bit integers as strings.
type number float64

func (value *number) UnmarshalJSON(data []byte) error {
	text := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if text == "" || text == "null" {
		return nil
	}
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return fmt.Errorf("%s is not a number", data)
	}
	*value = number(parsed)
	return nil
}

// decodeObject decodes a JSON object; an empty body is an empty object,
// since Google leaves out empty lists.
func decodeObject(payload []byte, into any) error {
	if len(strings.TrimSpace(string(payload))) == 0 {
		return nil
	}
	return json.Unmarshal(payload, into)
}

// parseTime reads an RFC 3339 timestamp as UTC; an empty value is no time.
func parseTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("%q is not a timestamp", value)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func finite(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	v := *value
	return &v
}
