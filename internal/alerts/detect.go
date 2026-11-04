package alerts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/gdacs"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigation"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/grid"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/push"
	"github.com/prabhavalabs/lanka-weather/internal/push/messages"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

// Hold times: how long a subject must stay below its alerted level before
// that level can alert again.
const (
	HoldBulletin   = 6 * time.Hour
	HoldLandslide  = 12 * time.Hour
	HoldRiver      = 6 * time.Hour
	HoldForecast   = 6 * time.Hour
	HoldCyclone    = 12 * time.Hour
	HoldEarthquake = 30 * 24 * time.Hour
	HoldReservoir  = 24 * time.Hour
)

// Thresholds of the hazards we measure rather than read from an issuer.
const (
	// CycloneNearKm is how close a cyclone's centre, or its forecast track,
	// must come to Sri Lanka's coast.
	CycloneNearKm = 1000
	// CycloneCloseKm makes a cyclone's centre this close a danger notice.
	CycloneCloseKm = 300
	// QuakeWindow is how recent an earthquake must be to be news.
	QuakeWindow = 6 * time.Hour
)

// Observation is one subject's level now, with the event to raise if it is
// alerted. Hold means the data cannot say (stale, unknown): nothing changes.
type Observation struct {
	Subject string
	Level   int
	Hold    bool
	// Event is the event for this level, without its key or detection time.
	Event push.Event
}

// Reading is what one source says now. Complete means a subject it does not
// mention is at level 0; otherwise unmentioned subjects are held.
type Reading struct {
	Kind         string
	HoldFor      time.Duration
	Complete     bool
	Observations []Observation
}

// shortHash names a subject whose natural key is long text.
func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:6])
}

func point(lon, lat float64) (*float64, *float64) { return &lon, &lat }

// levelSeverity maps a level 1 to 3 to watch, warning and danger.
func levelSeverity(level int) push.Severity {
	switch level {
	case 1:
		return push.SeverityWatch
	case 2:
		return push.SeverityWarning
	case 3:
		return push.SeverityDanger
	}
	return push.SeverityInfo
}

func lower(value *string) string {
	if value == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(*value))
}

func clampExpiry(until *time.Time, now time.Time, fallback, longest time.Duration) time.Time {
	expires := now.Add(fallback)
	if until != nil && until.After(now) {
		expires = *until
	}
	if limit := now.Add(longest); expires.After(limit) {
		expires = limit
	}
	return expires
}

// BulletinLevel is a bulletin's level: a red notice 3, a warning or an amber
// advisory 2, any other advisory 1.
func BulletinLevel(bulletin warnings.Bulletin) (int, string) {
	colour := lower(bulletin.Colour)
	switch {
	case colour == "red":
		return 3, "red"
	case bulletin.Severity == "warning":
		return 2, "warning"
	case colour == "amber":
		return 2, "advisory"
	}
	return 1, "advisory"
}

// Bulletins reads the DMC's warnings and advisories in force, one per warning
// (its Tamil edition and numbered updates are the same warning). Landslide
// reports are read by Landslides from NBRO's levels instead.
func Bulletins(overview *warnings.Overview, now time.Time) Reading {
	reading := Reading{Kind: messages.KindBulletin, HoldFor: HoldBulletin, Complete: !overview.Meta.Stale}
	for _, bulletin := range overview.InForce {
		if bulletin.Type == "landslide" || (bulletin.Severity != "warning" && bulletin.Severity != "advisory") {
			continue
		}
		level, variant := BulletinLevel(bulletin)
		key := shortHash(warnings.WarningKey(bulletin))
		districts := make([]string, 0, len(bulletin.Districts))
		for _, district := range bulletin.Districts {
			districts = append(districts, district.ID)
		}
		sort.Strings(districts)
		url := "/?warnings=" + bulletin.Type
		if len(districts) == 1 {
			url += "&district=" + districts[0]
		}
		reading.Observations = append(reading.Observations, Observation{
			Subject: "bulletin:" + key, Level: level,
			Event: push.Event{Kind: messages.KindBulletin, Variant: variant, Topic: push.TopicOfficial, Severity: levelSeverity(level),
				National: len(districts) == 0, Districts: districts,
				Facts: map[string]string{"title": bulletin.Title, "districts": strings.Join(districts, ","), "level": strconv.Itoa(level)},
				URL:   url, Tag: "bulletin:" + key, ExpiresAt: clampExpiry(bulletin.ValidUntil, now, 12*time.Hour, 24*time.Hour)},
		})
	}
	return reading
}

// Landslides reads NBRO's levels by district: the highest level of a district's
// DS divisions in the report in force. Only an active report raises levels and
// only an active or a "none" report lowers them; lapsed, pending and unknown
// say nothing either way.
func Landslides(layer warnings.LandslideLayer, now time.Time) Reading {
	reading := Reading{Kind: messages.KindLandslide, HoldFor: HoldLandslide}
	switch layer.Meta.Status {
	case warnings.StatusActive:
		reading.Complete = true
	case warnings.StatusNone:
		reading.Complete = true
		return reading
	default:
		return reading
	}
	highest, divisions := map[string]int{}, map[string]int{}
	for _, feature := range layer.Features {
		district := feature.Properties.DistrictID
		if district == "" {
			continue
		}
		highest[district] = max(highest[district], feature.Properties.Level)
		divisions[district]++
	}
	districts := make([]string, 0, len(highest))
	for district := range highest {
		districts = append(districts, district)
	}
	sort.Strings(districts)
	for _, district := range districts {
		level := highest[district]
		topic := push.TopicLandslide
		if level == 1 {
			topic = push.TopicLandslideLevel1
		}
		reading.Observations = append(reading.Observations, Observation{
			Subject: "landslide:" + district, Level: level,
			Event: push.Event{Kind: messages.KindLandslide, Variant: strconv.Itoa(level), Topic: topic, Severity: levelSeverity(level),
				Districts: []string{district},
				Facts:     map[string]string{"district": district, "level": strconv.Itoa(level), "divisions": strconv.Itoa(divisions[district])},
				URL:       "/?warnings=landslide&district=" + district, Tag: "landslide:district:" + district,
				ExpiresAt: clampExpiry(layer.Meta.ValidUntil, now, 24*time.Hour, 24*time.Hour)},
		})
	}
	return reading
}

// RiverLevel is a gauge status's level: alert 1, minor flood 2, major flood 3.
func RiverLevel(status rivers.Status) (int, string, bool) {
	switch status {
	case rivers.StatusNormal:
		return 0, "", true
	case rivers.StatusAlert:
		return 1, "alert", true
	case rivers.StatusMinorFlood:
		return 2, "minor_flood", true
	case rivers.StatusMajorFlood:
		return 3, "major_flood", true
	}
	return 0, "", false
}

// DistrictOf finds the district of a point; "" when none is within reach.
type DistrictOf func(lon, lat float64) string

// Rivers reads every gauge's flood status. A stale reading or a gauge with no
// thresholds is held. Gauges of the local test seed are ignored.
func Rivers(collection rivers.FeatureCollection, districtOf DistrictOf, now time.Time) Reading {
	reading := Reading{Kind: messages.KindRiver, HoldFor: HoldRiver, Complete: true}
	for _, feature := range collection.Features {
		station := feature.Properties
		if station.Source == floods.SeedSource {
			continue
		}
		subject := "river:" + station.ID
		level, variant, known := RiverLevel(station.Status)
		if !known || station.Stale {
			reading.Observations = append(reading.Observations, Observation{Subject: subject, Hold: true})
			continue
		}
		lon, lat := feature.Geometry.Coordinates[0], feature.Geometry.Coordinates[1]
		district := districtOf(lon, lat)
		source := "community"
		if station.Source == irrigation.SourceName {
			source = "irrigation"
		}
		event := push.Event{Kind: messages.KindRiver, Variant: variant, Topic: push.TopicRiver, Severity: levelSeverity(level),
			Facts: map[string]string{"station": station.Name, "river": station.Basin, "district": district, "source": source},
			URL:   "/?station=" + station.ID, Tag: "river:station:" + station.ID, ExpiresAt: now.Add(6 * time.Hour)}
		event.Lon, event.Lat = point(lon, lat)
		if district != "" {
			event.Districts = []string{district}
		}
		reading.Observations = append(reading.Observations, Observation{Subject: subject, Level: level, Event: event})
	}
	return reading
}

// ForecastLevel is a Google severity's level: Flood Hub's Warning 1, Danger
// 2 and Extreme 3.
func ForecastLevel(severity string) (int, string, bool) {
	switch googleflood.Severity(severity) {
	case googleflood.SeverityNoFlooding:
		return 0, "", true
	case googleflood.SeverityAboveNormal:
		return 1, "warning", true
	case googleflood.SeveritySevere:
		return 2, "danger", true
	case googleflood.SeverityExtreme:
		return 3, "extreme", true
	}
	return 0, "", false
}

// forecastSeverity keeps Google's forecasts at warning until Extreme: they are
// estimates, and only the top of their scale passes the hourly cap.
func forecastSeverity(level int) push.Severity {
	if level >= 3 {
		return push.SeverityDanger
	}
	return push.SeverityWarning
}

// FloodForecasts reads Google's verified gauges. Lower-confidence gauges never
// alert (ADR 0015), nor do stale or seeded ones.
func FloodForecasts(collection floods.FeatureCollection, districtOf DistrictOf, now time.Time) Reading {
	reading := Reading{Kind: messages.KindFloodForecast, HoldFor: HoldForecast, Complete: collection.Meta.Configured && collection.Meta.FetchedAt != nil}
	for _, feature := range collection.Features {
		gauge, ok := feature.Properties.(floods.Gauge)
		if !ok || !gauge.QualityVerified || gauge.Seeded {
			continue
		}
		subject := "forecast:" + gauge.GaugeID
		level, variant, known := ForecastLevel(gauge.Severity)
		if !known || gauge.Stale {
			reading.Observations = append(reading.Observations, Observation{Subject: subject, Hold: true})
			continue
		}
		district := districtOf(gauge.Lon, gauge.Lat)
		site := gauge.SiteName
		if site == "" {
			site = gauge.River
		}
		event := push.Event{Kind: messages.KindFloodForecast, Variant: variant, Topic: push.TopicFloodForecast, Severity: forecastSeverity(level),
			Facts: map[string]string{"site": site, "river": gauge.River, "district": district},
			URL:   "/?gauge=" + gauge.GaugeID, Tag: "forecast:gauge:" + gauge.GaugeID,
			ExpiresAt: clampExpiry(gauge.ForecastEnd, now, 12*time.Hour, 24*time.Hour)}
		event.Lon, event.Lat = point(gauge.Lon, gauge.Lat)
		if district != "" {
			event.Districts = []string{district}
		}
		reading.Observations = append(reading.Observations, Observation{Subject: subject, Level: level, Event: event})
	}
	return reading
}

// AreaDistricts finds the districts a flood area overlaps.
type AreaDistricts func(areaID string) []string

// FlashFloods reads Google's flash-flood areas over Sri Lanka (there have been
// none so far): likely 1, highly likely 2.
func FlashFloods(collection floods.FeatureCollection, areaDistricts AreaDistricts, now time.Time) Reading {
	reading := Reading{Kind: messages.KindFlashFlood, HoldFor: HoldForecast, Complete: collection.Meta.Configured && collection.Meta.FetchedAt != nil}
	if collection.Meta.Seeded {
		reading.Complete = false
		return reading
	}
	for _, feature := range collection.Features {
		area, ok := feature.Properties.(floods.Area)
		if !ok || (area.Kind != googleflood.KindFlashLikely && area.Kind != googleflood.KindFlashHighly) {
			continue
		}
		level, variant := 1, "likely"
		if area.Kind == googleflood.KindFlashHighly {
			level, variant = 2, "highly_likely"
		}
		districts := areaDistricts(area.ID)
		if len(districts) == 0 {
			continue
		}
		reading.Observations = append(reading.Observations, Observation{
			Subject: "flash:" + area.ID, Level: level,
			Event: push.Event{Kind: messages.KindFlashFlood, Variant: variant, Topic: push.TopicFloodForecast, Severity: push.SeverityWarning,
				Districts: districts, Facts: map[string]string{"district": districts[0], "districts": strings.Join(districts, ",")},
				URL: "/?area=" + area.ID, Tag: "flash:area:" + area.ID, ExpiresAt: clampExpiry(area.ValidUntil, now, 12*time.Hour, 24*time.Hour)},
		})
	}
	return reading
}

// geometryPoints lists the positions of a Point or LineString.
func geometryPoints(raw json.RawMessage) [][2]float64 {
	var geometry struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if json.Unmarshal(raw, &geometry) != nil {
		return nil
	}
	switch geometry.Type {
	case "Point":
		var position [2]float64
		if json.Unmarshal(geometry.Coordinates, &position) == nil {
			return [][2]float64{position}
		}
	case "LineString", "MultiPoint":
		var positions [][2]float64
		if json.Unmarshal(geometry.Coordinates, &positions) == nil {
			return positions
		}
	case "MultiLineString":
		var lines [][][2]float64
		if json.Unmarshal(geometry.Coordinates, &lines) == nil {
			out := [][2]float64{}
			for _, line := range lines {
				out = append(out, line...)
			}
			return out
		}
	}
	return nil
}

// Cyclones reads GDACS's current systems: centre within 300 km of the coast 3,
// within 1,000 km 2, forecast track within 1,000 km 1. A system whose last
// advisory is old is held.
func Cyclones(layer hazards.CycloneLayer, now time.Time) Reading {
	reading := Reading{Kind: messages.KindCyclone, HoldFor: HoldCyclone, Complete: !layer.Meta.Stale && layer.Meta.Status != hazards.CycloneWaiting}
	type system struct {
		name           string
		centreKm       float64
		forecastKm     float64
		stale, located bool
	}
	systems := map[string]*system{}
	get := func(id string) *system {
		if entry, ok := systems[id]; ok {
			return entry
		}
		entry := &system{centreKm: math.Inf(1), forecastKm: math.Inf(1)}
		systems[id] = entry
		return entry
	}
	for _, feature := range layer.Features {
		id, _ := feature.Properties["eventId"].(string)
		if id == "" {
			continue
		}
		entry := get(id)
		kind, _ := feature.Properties["kind"].(string)
		switch {
		case kind == "position":
			entry.name, _ = feature.Properties["name"].(string)
			entry.stale, _ = feature.Properties["stale"].(bool)
			if distance, ok := feature.Properties["distanceKm"].(float64); ok {
				entry.centreKm, entry.located = distance, true
			}
		case kind == gdacs.KindForecast || (kind == gdacs.KindPosition && feature.Properties["forecast"] == true):
			for _, position := range geometryPoints(feature.Geometry) {
				entry.forecastKm = math.Min(entry.forecastKm, grid.CoastDistanceKm(position[0], position[1]))
			}
		}
	}
	ids := make([]string, 0, len(systems))
	for id := range systems {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		entry := systems[id]
		subject := "cyclone:" + id
		if !entry.located || entry.stale {
			reading.Observations = append(reading.Observations, Observation{Subject: subject, Hold: true})
			continue
		}
		level, variant, distance := 0, "", entry.centreKm
		switch {
		case entry.centreKm <= CycloneCloseKm:
			level, variant = 3, "close"
		case entry.centreKm <= CycloneNearKm:
			level, variant = 2, "near"
		case entry.forecastKm <= CycloneNearKm:
			level, variant, distance = 1, "track", entry.forecastKm
		}
		reading.Observations = append(reading.Observations, Observation{
			Subject: subject, Level: level,
			Event: push.Event{Kind: messages.KindCyclone, Variant: variant, Topic: push.TopicCyclone, Severity: levelSeverity(level), National: true,
				Facts: map[string]string{"name": entry.name, "distance": strconv.Itoa(int(math.Round(distance)))},
				URL:   "/?cyclone=" + id, Tag: "cyclone:" + id, ExpiresAt: now.Add(12 * time.Hour)},
		})
	}
	return reading
}

// QuakeLevel says how much an earthquake matters here. Level 2, a large
// earthquake whose notice refers people to the DMC for tsunami advice: M6.5
// within 1,500 km of the coast, M7 anywhere in the region USGS is asked about
// (15° S to 25° N, 60 to 105° E), or USGS's tsunami flag. Level 1: M5 within
// 500 km, M6 in the region, or M4 within 300 km, which may be felt here.
func QuakeLevel(magnitude, distanceKm float64, tsunami bool) int {
	switch {
	case magnitude >= 7, magnitude >= 6.5 && distanceKm <= 1500, tsunami && magnitude >= 6:
		return 2
	case magnitude >= 6, magnitude >= 5 && distanceKm <= 500, magnitude >= 4 && distanceKm <= 300:
		return 1
	}
	return 0
}

// Earthquakes reads USGS's events of the last six hours. An event that leaves
// the list (deleted, or older) falls to level 0.
func Earthquakes(layer hazards.EarthquakeLayer, now time.Time) Reading {
	reading := Reading{Kind: messages.KindEarthquake, HoldFor: HoldEarthquake, Complete: !layer.Meta.Stale}
	for _, feature := range layer.Features {
		quake := feature.Properties
		if now.Sub(quake.Time) > QuakeWindow {
			continue
		}
		level := QuakeLevel(quake.Magnitude, quake.DistanceKm, quake.Tsunami)
		if level == 0 {
			continue
		}
		variant := "plain"
		if level == 2 {
			variant = "major"
		}
		reading.Observations = append(reading.Observations, Observation{
			Subject: "quake:" + quake.ID, Level: level,
			Event: push.Event{Kind: messages.KindEarthquake, Variant: variant, Topic: push.TopicEarthquake, Severity: levelSeverity(level), National: true,
				Facts: map[string]string{"magnitude": messages.Magnitude(quake.Magnitude), "place": quake.Place,
					"distance": strconv.Itoa(int(math.Round(quake.DistanceKm)))},
				URL: "/?quake=" + quake.ID, Tag: "quake:" + quake.ID, ExpiresAt: quake.Time.Add(QuakeWindow)},
		})
	}
	return reading
}

// DistrictNamed finds a district's code from its English name.
type DistrictNamed func(name string) string

// Reservoirs reads which reservoirs are spilling, on current readings only,
// for the districts people chose. A reservoir with no known district is left
// out. These come from waterlevels.lk and are not official warnings.
func Reservoirs(collection reservoirs.FeatureCollection, districtNamed DistrictNamed, now time.Time) Reading {
	reading := Reading{Kind: messages.KindReservoir, HoldFor: HoldReservoir, Complete: collection.Meta.FetchedAt != nil}
	for _, feature := range collection.Features {
		properties := feature.Properties
		if properties.District == nil {
			continue
		}
		district := districtNamed(*properties.District)
		if district == "" {
			continue
		}
		subject := "reservoir:" + properties.ID
		if properties.Stale {
			reading.Observations = append(reading.Observations, Observation{Subject: subject, Hold: true})
			continue
		}
		level := 0
		if properties.Status == waterlevels.StatusSpilling {
			level = 1
		}
		reading.Observations = append(reading.Observations, Observation{
			Subject: subject, Level: level,
			Event: push.Event{Kind: messages.KindReservoir, Variant: "spill", Topic: push.TopicReservoir, Severity: push.SeverityWatch,
				Districts: []string{district}, Facts: map[string]string{"name": properties.Name, "district": district},
				URL: "/?reservoir=" + properties.ID, Tag: "reservoir:" + properties.ID, ExpiresAt: now.Add(24 * time.Hour)},
		})
	}
	return reading
}

// normaliseName compares district names whatever their spacing and case.
func normaliseName(name string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' {
			out.WriteRune(r)
		}
	}
	return out.String()
}
