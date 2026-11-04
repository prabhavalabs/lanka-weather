// Package stations serves Department of Meteorology observations, and rainfall
// totals at those stations and at Irrigation Department gauges. Everything here
// is observed rather than modelled; a reading always carries its time, and an
// old one is marked stale rather than shown as current. It never calls a
// source for a request.
package stations

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/metstations"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// ErrStationNotFound is returned for an unknown station identifier.
var ErrStationNotFound = errors.New("station not found")

// How old a reading may be before it is marked stale, for each kind. A daily
// report is issued once a day, so a day and a half allows for a late one.
const (
	DailyStaleAfter = 36 * time.Hour
	GaugeStaleAfter = 3 * time.Hour
)

const (
	// rainMaxAge leaves out totals too old to say anything about recent rain.
	rainMaxAge = 7 * 24 * time.Hour
	// nearbyKm bounds the nearest station a place is told about.
	nearbyKm = 75.0
	// minimumSlots of the eight 3-hourly readings make a 24-hour total.
	minimumSlots = 6
	// minimumRangeReadings is how many readings give a day's temperature range.
	minimumRangeReadings = 4
	// Gauges report about hourly. A total needs most of the day covered.
	minimumGaugeReadings = 18
	gaugeLookback        = 48 * time.Hour
	gaugeWindow          = 24 * time.Hour
	// A gauge reading entered again within this time, with the same level and
	// rain, counts once.
	gaugeRepeat = 10 * time.Minute
)

// Kinds of reading.
const (
	ReadingThreeHourly = "3h"
	ReadingDaily       = "daily"
	ReadingNone        = "none"
)

// Service reads observations from the database.
type Service struct {
	Pool *pgxpool.Pool
	// StaleAfter is the age after which a 3-hourly reading is marked stale.
	StaleAfter time.Duration
	Now        func() time.Time
}

// FeatureCollection is the weather-stations layer as GeoJSON.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
	Meta     Meta      `json:"meta"`
}

// Meta describes the layer's freshness and origin.
type Meta struct {
	Attribution string    `json:"attribution"`
	SourceURL   string    `json:"sourceUrl"`
	GeneratedAt time.Time `json:"generatedAt"`
	// Observed marks this as measurements, unlike the model grids.
	Observed    bool `json:"observed"`
	ThreeHourly Feed `json:"threeHourly"`
	Daily       Feed `json:"daily"`
	// Stale is true when neither feed is current.
	Stale bool `json:"stale"`
}

// Feed is one source's freshness.
type Feed struct {
	Source string `json:"source"`
	// NewestObservedAt is the newest reading the source has given; FetchedAt
	// is when we last stored something new from it.
	NewestObservedAt *time.Time `json:"newestObservedAt"`
	FetchedAt        *time.Time `json:"fetchedAt"`
	Stale            bool       `json:"stale"`
}

// Point is a GeoJSON point in longitude, latitude order.
type Point struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

// Feature is one station.
type Feature struct {
	Type       string     `json:"type"`
	ID         string     `json:"id"`
	Geometry   Point      `json:"geometry"`
	Properties Properties `json:"properties"`
}

// Properties are a station's details and its latest reading.
type Properties struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	WMOID      *string  `json:"wmoId"`
	ElevationM *float64 `json:"elevationM"`
	// PositionApproximate marks a position geocoded from the name.
	PositionApproximate bool `json:"positionApproximate"`
	// Reading says which feed the values come from: "3h", "daily" or "none".
	Reading    string     `json:"reading"`
	ObservedAt *time.Time `json:"observedAt"`
	// TemperatureC is an instantaneous reading, from the 3-hourly feed only.
	TemperatureC *float64 `json:"temperatureC"`
	HumidityPct  *float64 `json:"humidityPct"`
	// RainMm fell in the RainHours ending at ObservedAt.
	RainMm    *float64 `json:"rainMm"`
	RainHours int      `json:"rainHours"`
	// Rain24hMm is over the 24 hours ending at ObservedAt, where computable.
	Rain24hMm *float64 `json:"rain24hMm"`
	// TempMinC and TempMaxC are the daily report's, or the range of the last
	// 24 hours of 3-hourly readings.
	TempMinC *float64 `json:"tempMinC"`
	TempMaxC *float64 `json:"tempMaxC"`
	// ReportsTemperature is false for a rain-only station.
	ReportsTemperature bool `json:"reportsTemperature"`
	Stale              bool `json:"stale"`
}

// Limits are the ages after which readings are marked stale.
type Limits struct{ ThreeHourly, Daily time.Duration }

// Three is a 3-hourly reading.
type Three struct {
	ObservedAt         time.Time
	RainMm, TempC, Hum *float64
}

// Day is a daily report.
type Day struct {
	ObservedAt       time.Time
	RainMm, Min, Max *float64
}

// Latest chooses what a station shows. A current 3-hourly reading wins. When
// it is out of date the daily report takes over if that is newer; otherwise
// the newest reading there is stays, marked stale. `readings` covers the 24
// hours to the newest 3-hourly reading, newest first.
func Latest(properties Properties, readings []Three, day *Day, now time.Time, limits Limits) Properties {
	properties.Reading = ReadingNone
	properties.Stale = true
	var newest *Three
	if len(readings) > 0 {
		newest = &readings[0]
	}
	switch {
	case newest != nil && (now.Sub(newest.ObservedAt) <= limits.ThreeHourly || day == nil || !day.ObservedAt.After(newest.ObservedAt)):
		observed := newest.ObservedAt
		properties.Reading, properties.ObservedAt = ReadingThreeHourly, &observed
		properties.TemperatureC, properties.HumidityPct, properties.RainMm = newest.TempC, newest.Hum, newest.RainMm
		properties.RainHours = 3
		properties.Rain24hMm = sumRain(readings)
		properties.TempMinC, properties.TempMaxC = temperatureRange(readings)
		properties.Stale = now.Sub(observed) > limits.ThreeHourly
	case day != nil:
		observed := day.ObservedAt
		properties.Reading, properties.ObservedAt = ReadingDaily, &observed
		properties.RainMm, properties.Rain24hMm, properties.RainHours = day.RainMm, day.RainMm, 24
		properties.TempMinC, properties.TempMaxC = day.Min, day.Max
		properties.Stale = now.Sub(observed) > limits.Daily
	}
	properties.ReportsTemperature = properties.TemperatureC != nil || properties.TempMaxC != nil || properties.TempMinC != nil
	return properties
}

// sumRain totals the rain of the readings, which cover the day to the newest,
// when enough of the eight are there to call it a day's rain.
func sumRain(readings []Three) *float64 {
	total, counted := 0.0, 0
	for _, reading := range readings {
		if reading.RainMm != nil {
			total += *reading.RainMm
			counted++
		}
	}
	if counted < minimumSlots {
		return nil
	}
	total = round1(total)
	return &total
}

func temperatureRange(readings []Three) (low, high *float64) {
	counted := 0
	for _, reading := range readings {
		if reading.TempC == nil {
			continue
		}
		counted++
		if low == nil || *reading.TempC < *low {
			low = reading.TempC
		}
		if high == nil || *reading.TempC > *high {
			high = reading.TempC
		}
	}
	if counted < minimumRangeReadings {
		return nil, nil
	}
	return low, high
}

func (service *Service) limits() Limits {
	return Limits{ThreeHourly: service.StaleAfter, Daily: DailyStaleAfter}
}

// Layer returns every station with its latest reading.
func (service *Service) Layer(ctx context.Context) (FeatureCollection, error) {
	now := service.now()
	queries := store.New(service.Pool)
	stations, err := queries.ListMetStations(ctx)
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("list met stations: %w", err)
	}
	threeRows, err := queries.LatestMetReadings3h(ctx)
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("read 3-hourly readings: %w", err)
	}
	dayRows, err := queries.LatestMetReadingsDaily(ctx)
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("read daily readings: %w", err)
	}
	freshness, err := queries.MetFeedFreshness(ctx)
	if err != nil {
		return FeatureCollection{}, fmt.Errorf("read feed freshness: %w", err)
	}
	threes := map[string][]Three{}
	for _, row := range threeRows {
		threes[row.StationID] = append(threes[row.StationID], Three{ObservedAt: row.ObservedAt.UTC(), RainMm: row.RainMm, TempC: row.TempC, Hum: row.HumidityPct})
	}
	days := map[string]*Day{}
	for _, row := range dayRows {
		days[row.StationID] = &Day{ObservedAt: row.ObservedAt.UTC(), RainMm: row.RainMm, Min: row.TempMinC, Max: row.TempMaxC}
	}
	limits := service.limits()
	collection := FeatureCollection{Type: "FeatureCollection", Features: make([]Feature, 0, len(stations)),
		Meta: BuildMeta(freshness, now, limits)}
	for _, station := range stations {
		properties := Latest(propertiesOf(station.ID, station.Name, station.WmoID, station.ElevationM, station.PositionSource),
			threes[station.ID], days[station.ID], now, limits)
		collection.Features = append(collection.Features, Feature{Type: "Feature", ID: station.ID,
			Geometry:   Point{Type: "Point", Coordinates: [2]float64{station.Lon, station.Lat}},
			Properties: properties})
	}
	return collection, nil
}

func propertiesOf(id, name string, wmo *string, elevation *float64, positionSource string) Properties {
	return Properties{ID: id, Name: name, WMOID: wmo, ElevationM: elevation,
		PositionApproximate: positionSource == metstations.PositionGeocoded}
}

// BuildMeta describes the two feeds' freshness. A feed that has never given
// anything is stale.
func BuildMeta(freshness store.MetFeedFreshnessRow, now time.Time, limits Limits) Meta {
	feed := func(source string, newest, fetched time.Time, limit time.Duration) Feed {
		out := Feed{Source: source, Stale: true}
		if newest.Unix() <= 0 {
			return out
		}
		newest, fetched = newest.UTC(), fetched.UTC()
		out.NewestObservedAt, out.FetchedAt = &newest, &fetched
		out.Stale = now.Sub(newest) > limit
		return out
	}
	three := feed(metstations.SourceThreeHourly, freshness.Newest3h, freshness.Fetched3h, limits.ThreeHourly)
	daily := feed(metstations.SourceDaily, freshness.NewestDaily, freshness.FetchedDaily, limits.Daily)
	return Meta{Attribution: metstations.Attribution, SourceURL: "https://www.meteo.gov.lk", GeneratedAt: now,
		Observed: true, ThreeHourly: three, Daily: daily, Stale: three.Stale && daily.Stale}
}

// Detail is GET /v1/stations/{id}.
type Detail struct {
	Station Feature `json:"station"`
	// ThreeHourly and Daily are oldest first.
	ThreeHourly []ThreeHourlyReading `json:"threeHourly"`
	Daily       []DailyReading       `json:"daily"`
	Meta        Meta                 `json:"meta"`
}

// ThreeHourlyReading is one reading of a station's history.
type ThreeHourlyReading struct {
	ObservedAt   time.Time `json:"observedAt"`
	TemperatureC *float64  `json:"temperatureC"`
	HumidityPct  *float64  `json:"humidityPct"`
	RainMm       *float64  `json:"rainMm"`
}

// DailyReading is one daily report of a station's history.
type DailyReading struct {
	Date       string    `json:"date"`
	ObservedAt time.Time `json:"observedAt"`
	RainMm     *float64  `json:"rainMm"`
	TempMinC   *float64  `json:"tempMinC"`
	TempMaxC   *float64  `json:"tempMaxC"`
}

// Station returns one station's latest reading and its readings over a period
// ending at the newest reading. A source that has stopped would otherwise
// show an empty history.
func (service *Service) Station(ctx context.Context, id string, days int) (*Detail, error) {
	queries := store.New(service.Pool)
	station, err := queries.GetMetStation(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read station: %w", err)
	}
	layer, err := service.Layer(ctx)
	if err != nil {
		return nil, err
	}
	detail := &Detail{ThreeHourly: []ThreeHourlyReading{}, Daily: []DailyReading{}, Meta: layer.Meta}
	for _, feature := range layer.Features {
		if feature.ID == station.ID {
			detail.Station = feature
		}
	}
	period := time.Duration(days) * 24 * time.Hour
	since := func(feed Feed) time.Time {
		if feed.NewestObservedAt == nil {
			return service.now().Add(-period)
		}
		return feed.NewestObservedAt.Add(-period)
	}
	threes, err := queries.StationMetReadings3h(ctx, store.StationMetReadings3hParams{StationID: id, ObservedAt: since(layer.Meta.ThreeHourly)})
	if err != nil {
		return nil, fmt.Errorf("read 3-hourly history: %w", err)
	}
	for _, row := range threes {
		detail.ThreeHourly = append(detail.ThreeHourly, ThreeHourlyReading{ObservedAt: row.ObservedAt.UTC(), TemperatureC: row.TempC, HumidityPct: row.HumidityPct, RainMm: row.RainMm})
	}
	dailies, err := queries.StationMetReadingsDaily(ctx, store.StationMetReadingsDailyParams{StationID: id, ObservedAt: since(layer.Meta.Daily).Add(-30 * 24 * time.Hour)})
	if err != nil {
		return nil, fmt.Errorf("read daily history: %w", err)
	}
	for _, row := range dailies {
		detail.Daily = append(detail.Daily, DailyReading{Date: row.ReportDate.Time.Format(time.DateOnly), ObservedAt: row.ObservedAt.UTC(), RainMm: row.RainMm, TempMinC: row.TempMinC, TempMaxC: row.TempMaxC})
	}
	return detail, nil
}

// Nearby is the observed reading nearest a place.
type Nearby struct {
	Properties
	DistanceKm  float64 `json:"distanceKm"`
	Lon         float64 `json:"lon"`
	Lat         float64 `json:"lat"`
	Attribution string  `json:"attribution"`
}

// Nearest finds the closest station with a temperature, within 75 km.
func (service *Service) Nearest(ctx context.Context, lon, lat float64) (*Nearby, error) {
	layer, err := service.Layer(ctx)
	if err != nil {
		return nil, err
	}
	return NearestIn(layer, lon, lat), nil
}

// NearestIn finds the closest station with a temperature in a layer.
func NearestIn(layer FeatureCollection, lon, lat float64) *Nearby {
	var best *Nearby
	for _, feature := range layer.Features {
		if !feature.Properties.ReportsTemperature || feature.Properties.Reading == ReadingNone {
			continue
		}
		distance := distanceKm(lon, lat, feature.Geometry.Coordinates[0], feature.Geometry.Coordinates[1])
		if distance > nearbyKm || (best != nil && distance >= best.DistanceKm) {
			continue
		}
		best = &Nearby{Properties: feature.Properties, DistanceKm: round1(distance), Lon: feature.Geometry.Coordinates[0],
			Lat: feature.Geometry.Coordinates[1], Attribution: layer.Meta.Attribution}
	}
	return best
}

// RainCollection is GET /v1/rain/observed.
type RainCollection struct {
	Type     string        `json:"type"`
	Features []RainFeature `json:"features"`
	Meta     RainMeta      `json:"meta"`
}

// RainMeta describes the observed totals.
type RainMeta struct {
	Attribution string    `json:"attribution"`
	GeneratedAt time.Time `json:"generatedAt"`
	Observed    bool      `json:"observed"`
	// Fresh counts the points whose total is current.
	Fresh int `json:"fresh"`
	Stale int `json:"stale"`
}

// RainFeature is one point with a 24-hour total.
type RainFeature struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Geometry   Point          `json:"geometry"`
	Properties RainProperties `json:"properties"`
}

// Kinds of rainfall point and how a total was reached.
const (
	KindMet   = "met"
	KindGauge = "gauge"

	MethodDailyReport = "daily_report"
	MethodSum3h       = "sum_3h"
	MethodSumGauge    = "sum_gauge"
)

// RainProperties describe a 24-hour total.
type RainProperties struct {
	ID string `json:"id"`
	// StationID opens the station: a Met station, or a river gauge.
	StationID string  `json:"stationId"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	RainMm    float64 `json:"rainMm"`
	// WindowEnd closes the 24 hours the total covers.
	WindowEnd time.Time `json:"windowEnd"`
	Method    string    `json:"method"`
	Source    string    `json:"source"`
	Stale     bool      `json:"stale"`
}

// GaugeReading is one river gauge's rainfall reading.
type GaugeReading struct {
	ObservedAt time.Time
	WaterLevel float64
	RainMm     *float64
}

// GaugeTotal sums a gauge's rain over the 24 hours to its newest reading.
// Each reading holds the rain since the one before it. `readings` are newest
// first. It returns false when too few readings cover the day to call it a
// total, so a station that reports rarely is not shown as dry.
func GaugeTotal(readings []GaugeReading) (total float64, end time.Time, ok bool) {
	if len(readings) == 0 {
		return 0, time.Time{}, false
	}
	end = readings[0].ObservedAt
	counted := 0
	var previous *GaugeReading
	for _, reading := range readings {
		if end.Sub(reading.ObservedAt) >= gaugeWindow {
			break
		}
		// The same reading entered twice counts once. Readings moments apart that
		// differ are a backlog delivered together, and each holds its own rain.
		if previous != nil && previous.ObservedAt.Sub(reading.ObservedAt) < gaugeRepeat && sameReading(*previous, reading) {
			continue
		}
		current := reading
		previous = &current
		if reading.RainMm == nil {
			continue
		}
		total += *reading.RainMm
		counted++
	}
	if counted < minimumGaugeReadings {
		return 0, time.Time{}, false
	}
	return total, end, true
}

func sameReading(a, b GaugeReading) bool {
	sameRain := (a.RainMm == nil && b.RainMm == nil) || (a.RainMm != nil && b.RainMm != nil && *a.RainMm == *b.RainMm)
	return sameRain && a.WaterLevel == b.WaterLevel
}

// ObservedRainfall returns 24-hour totals at stations and river gauges.
func (service *Service) ObservedRainfall(ctx context.Context) (*RainCollection, error) {
	now := service.now()
	layer, err := service.Layer(ctx)
	if err != nil {
		return nil, err
	}
	queries := store.New(service.Pool)
	gauges, err := queries.ListRiverStations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list gauges: %w", err)
	}
	rows, err := queries.RecentRiverReadings(ctx, now.Add(-gaugeLookback))
	if err != nil {
		return nil, fmt.Errorf("read gauge readings: %w", err)
	}
	readings := map[string][]GaugeReading{}
	for _, row := range rows {
		readings[row.StationID] = append(readings[row.StationID], GaugeReading{ObservedAt: row.ObservedAt.UTC(), WaterLevel: row.WaterLevel, RainMm: row.RainfallMm})
	}
	out := &RainCollection{Type: "FeatureCollection", Features: []RainFeature{},
		Meta: RainMeta{Attribution: "Department of Meteorology, Sri Lanka; Irrigation Department of Sri Lanka", GeneratedAt: now, Observed: true}}
	add := func(feature RainFeature) {
		if now.Sub(feature.Properties.WindowEnd) > rainMaxAge {
			return
		}
		if feature.Properties.Stale {
			out.Meta.Stale++
		} else {
			out.Meta.Fresh++
		}
		out.Features = append(out.Features, feature)
	}
	for _, feature := range layer.Features {
		properties := feature.Properties
		if properties.Rain24hMm == nil || properties.ObservedAt == nil {
			continue
		}
		method, source := MethodSum3h, metstations.SourceThreeHourly
		if properties.Reading == ReadingDaily {
			method, source = MethodDailyReport, metstations.SourceDaily
		}
		add(RainFeature{Type: "Feature", ID: "met:" + properties.ID, Geometry: feature.Geometry, Properties: RainProperties{
			ID: "met:" + properties.ID, StationID: properties.ID, Kind: KindMet, Name: properties.Name, RainMm: round1(*properties.Rain24hMm),
			WindowEnd: *properties.ObservedAt, Method: method, Source: source, Stale: properties.Stale,
		}})
	}
	for _, gauge := range gauges {
		total, end, ok := GaugeTotal(readings[gauge.ID])
		if !ok {
			continue
		}
		add(RainFeature{Type: "Feature", ID: "gauge:" + gauge.ID, Geometry: Point{Type: "Point", Coordinates: [2]float64{gauge.Lon, gauge.Lat}},
			Properties: RainProperties{ID: "gauge:" + gauge.ID, StationID: gauge.ID, Kind: KindGauge, Name: gauge.Name, RainMm: round1(total),
				WindowEnd: end, Method: MethodSumGauge, Source: irrigation.SourceName, Stale: now.Sub(end) > GaugeStaleAfter}})
	}
	sort.Slice(out.Features, func(i, j int) bool { return out.Features[i].ID < out.Features[j].ID })
	return out, nil
}

func round1(value float64) float64 { return math.Round(value*10) / 10 }

// distanceKm is the great-circle distance between two points.
func distanceKm(lon1, lat1, lon2, lat2 float64) float64 {
	const earthRadiusKm = 6371.0088
	toRadians := math.Pi / 180
	dLat, dLon := (lat2-lat1)*toRadians, (lon2-lon1)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
