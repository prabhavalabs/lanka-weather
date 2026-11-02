// Package overview composes "Sri Lanka now": what the portal's other services
// say, gathered for a first look. It decides nothing about the hazards. Official
// warnings pass through in their issuers' words; estimates stay labelled; and each
// part carries its own age, source and the panel it opens. Build and Sentence are
// pure, and use no model of any kind: the status is assembled by fixed rules.
package overview

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/dmcbulletins"
	"github.com/prabhavalabs/lanka-weather/internal/environment"
	"github.com/prabhavalabs/lanka-weather/internal/hazards"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/stations"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// Section ids, in the order they are shown.
const (
	SectionLandslides  = "landslides"
	SectionWarnings    = "warnings"
	SectionGauges      = "gauges"
	SectionRivers      = "rivers"
	SectionReservoirs  = "reservoirs"
	SectionCyclones    = "cyclones"
	SectionEarthquakes = "earthquakes"
	SectionWeather     = "weather"
	SectionRain        = "rain"
	SectionAir         = "air"
	SectionSea         = "sea"
)

// Groups sections are shown under.
const (
	GroupOfficial = "official"
	GroupWater    = "water"
	GroupHazards  = "hazards"
	GroupNow      = "now"
)

// Tones say how much attention a line asks for. Colour is never the only sign:
// every tone comes with words.
const (
	ToneNeutral = "neutral"
	ToneWatch   = "watch"
	ToneHigh    = "high"
	ToneSevere  = "severe"
)

// Target says what a line opens.
type Target struct {
	// Kind is warnings, landslide, river, station, reservoir, reservoirs, cyclone,
	// quake, region, place or layer.
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
	// Reach is a stretch of a river.
	Reach *int64 `json:"reach,omitempty"`
	// Layer is switched on first.
	Layer string `json:"layer,omitempty"`
	// Fill is what to colour a region by: temperature, rain or wind.
	Fill string `json:"fill,omitempty"`
	// Lon and Lat are a place to show.
	Lon *float64 `json:"lon,omitempty"`
	Lat *float64 `json:"lat,omitempty"`
}

// Item is one line of a section.
type Item struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	// Tone and Badge are absent on a line that needs neither.
	Tone  string `json:"tone"`
	Badge string `json:"badge,omitempty"`
	// Estimate marks a derived or modelled figure.
	Estimate bool `json:"estimate"`
	// Stale says this line's own reading is out of date.
	Stale bool `json:"stale"`
	// At is how old the line is.
	At     *time.Time `json:"at"`
	Source string     `json:"source"`
	Target *Target    `json:"target"`
	// Href is the official document, when there is one.
	Href *string `json:"href,omitempty"`
}

// Count is one figure in a section's tally.
type Count struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value int    `json:"value"`
}

// Section is one part of the overview.
type Section struct {
	ID    string `json:"id"`
	Group string `json:"group"`
	Title string `json:"title"`
	// Summary is the section in a sentence.
	Summary string `json:"summary"`
	// Note says what the section is and is not.
	Note string `json:"note,omitempty"`
	// Official marks the issuers' own words and levels.
	Official bool `json:"official"`
	Estimate bool `json:"estimate"`
	// Available is false when the part could not be read; Stale when what was
	// read is too old to speak for today. Neither may read as calm.
	Available bool       `json:"available"`
	Stale     bool       `json:"stale"`
	At        *time.Time `json:"at"`
	Source    string     `json:"source"`
	Counts    []Count    `json:"counts"`
	Items     []Item     `json:"items"`
	Target    *Target    `json:"target"`
}

// Overview is GET /v1/overview.
type Overview struct {
	GeneratedAt time.Time `json:"generatedAt"`
	// Status is one sentence built from fixed rules (see Sentence).
	Status string `json:"status"`
	// Stale names the feeds too old to count, as the status says them.
	Stale []string `json:"stale"`
	// Unavailable names the feeds that could not be read.
	Unavailable []string  `json:"unavailable"`
	Sections    []Section `json:"sections"`
}

// Inputs is what the other services said; nil when one could not be read.
type Inputs struct {
	Warnings    *warnings.Overview
	Landslides  *warnings.LandslideLayer
	Risk        *riverrisk.RiskDocument
	Gauges      *rivers.FeatureCollection
	Reservoirs  *reservoirs.FeatureCollection
	Cyclones    *hazards.CycloneLayer
	Earthquakes *hazards.EarthquakeLayer
	Weather     *weather.RegionsResponse
	// WeatherStale says the region weather is too old to speak for now.
	WeatherStale bool
	Rain         *stations.RainCollection
	Air          *AirWorst
	Sea          *SeaHighest
}

// AirWorst is the highest air quality index on the grid, and where.
type AirWorst struct {
	environment.AirPoint
	Lon, Lat float64
	// Place names the district it falls in, or "" if none.
	Place string
}

// SeaHighest is the highest wave height on the grid, and where.
type SeaHighest struct {
	environment.SeaPoint
	Lon, Lat float64
}

const (
	maxWorstRivers  = 5
	maxDivisions    = 5
	maxWettest      = 3
	maxRainGauges   = 3
	maxQuakes       = 3
	significantDays = 7
	// A quake is worth a line if it is large anywhere in the list, moderate and
	// within reach of the Indian Ocean's nearer plate boundaries, near at any
	// size, or if USGS flags a tsunami. Most of the 30-day list is neither.
	quakeLargeM      = 6.0
	quakeModerateM   = 5.0
	quakeModerateKm  = 1500
	quakeNearKm      = 300
	sourceLandslides = "NBRO via the DMC"
	sourceBulletins  = "Disaster Management Centre"
	sourceGauges     = "Irrigation Department"
	sourceRisk       = "Estimate by lanka-weather"
	sourceReservoirs = "Irrigation Department and Mahaweli Authority via waterlevels.lk"
	sourceCyclones   = "GDACS"
	sourceQuakes     = "USGS"
	sourceWeather    = "Open-Meteo"
	sourceAir        = "CAMS via Open-Meteo"
	sourceSea        = "Open-Meteo marine model"
	// spillNoticeCusecs is the figure below which a spill is a trickle: the river
	// estimate's own line (ADR 0010).
	spillNoticeCusecs = 100
	// reservoirNote is said wherever reservoir figures are.
	reservoirNote = "Not an official warning. In an emergency call 117."
)

// plural writes a count with its noun.
func plural(count int, one, many string) string {
	if count == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", count, many)
}

func target(kind, id string) *Target { return &Target{Kind: kind, ID: id} }

func pointer[T any](value T) *T { return &value }

// Build composes the overview. Anything nil is listed as unavailable and says so
// in its section, never as calm.
func Build(in Inputs, now time.Time) *Overview {
	sections := []Section{
		landslideSection(in),
		warningsSection(in),
		gaugesSection(in),
		riversSection(in),
		reservoirsSection(in),
		cyclonesSection(in),
		earthquakesSection(in, now),
		weatherSection(in),
		rainSection(in),
		airSection(in),
		seaSection(in),
	}
	overview := &Overview{GeneratedAt: now, Stale: []string{}, Unavailable: []string{}, Sections: sections}
	for _, section := range sections {
		if !section.Available {
			overview.Unavailable = append(overview.Unavailable, feedNames[section.ID])
		} else if section.Stale {
			overview.Stale = append(overview.Stale, feedNames[section.ID])
		}
	}
	overview.Status = Sentence(sections, now)
	return overview
}

// feedNames are the feeds' names in the status.
var feedNames = map[string]string{
	SectionLandslides: "landslide levels", SectionWarnings: "official warnings", SectionGauges: "river gauges",
	SectionRivers: "river estimates", SectionReservoirs: "reservoirs", SectionCyclones: "cyclones",
	SectionEarthquakes: "earthquakes", SectionWeather: "weather", SectionRain: "gauge rainfall",
	SectionAir: "air quality", SectionSea: "sea conditions",
}

func unavailable(id, group, title, source string, official, estimate bool, into *Target) Section {
	return Section{ID: id, Group: group, Title: title, Source: source, Official: official, Estimate: estimate,
		Summary: "Could not be read just now, so nothing can be said from here.",
		Counts:  []Count{}, Items: []Item{}, Target: into}
}

func landslideSection(in Inputs) Section {
	into := target("warnings", "")
	if in.Landslides == nil {
		section := unavailable(SectionLandslides, GroupOfficial, "Landslide warnings", sourceLandslides, true, false, into)
		section.Note = "Check with the Disaster Management Centre."
		return section
	}
	meta := in.Landslides.Meta
	section := Section{ID: SectionLandslides, Group: GroupOfficial, Title: "Landslide warnings", Official: true, Available: true,
		Stale: meta.Stale, At: firstTime(meta.IssuedAt, meta.FetchedAt), Source: sourceLandslides, Summary: meta.Note, Note: meta.Limit,
		Counts: []Count{}, Items: []Item{}, Target: into}
	shown := map[int]int{}
	features := append([]warnings.DivisionFeature(nil), in.Landslides.Features...)
	sort.SliceStable(features, func(i, j int) bool { return features[i].Properties.Level > features[j].Properties.Level })
	for _, feature := range features {
		shown[feature.Properties.Level]++
	}
	for _, level := range []int{3, 2, 1} {
		words := warnings.WordsFor(level)
		section.Counts = append(section.Counts, Count{Key: fmt.Sprintf("level%d", level), Label: words.Label, Value: shown[level]})
	}
	for index, feature := range features {
		if index == maxDivisions {
			break
		}
		properties := feature.Properties
		section.Items = append(section.Items, Item{ID: properties.ID, Title: properties.Name,
			Detail: fmt.Sprintf("%s District. NBRO: %s.", properties.District, properties.Action),
			Tone:   landslideTone(properties.Level), Badge: properties.Label, At: section.At, Stale: meta.Stale,
			Source: sourceLandslides, Target: target("landslide", properties.ID), Href: meta.PDFURL})
	}
	return section
}

func landslideTone(level int) string {
	switch level {
	case 3:
		return ToneSevere
	case 2:
		return ToneHigh
	case 1:
		return ToneWatch
	}
	return ToneNeutral
}

func warningsSection(in Inputs) Section {
	into := target("warnings", "")
	if in.Warnings == nil {
		section := unavailable(SectionWarnings, GroupOfficial, "Warnings in force", sourceBulletins, true, false, into)
		section.Note = "Check with the Disaster Management Centre and the Department of Meteorology."
		return section
	}
	meta := in.Warnings.Meta
	section := Section{ID: SectionWarnings, Group: GroupOfficial, Title: "Warnings in force", Official: true, Available: true,
		Stale: meta.Stale, At: meta.FetchedAt, Source: sourceBulletins, Note: meta.Note,
		Counts: []Count{{Key: "inForce", Label: "In force", Value: len(in.Warnings.InForce)}}, Items: []Item{}, Target: into}
	switch {
	case len(in.Warnings.InForce) > 0:
		section.Summary = plural(len(in.Warnings.InForce), "warning or advisory", "warnings and advisories") + " in force, in the issuers' words."
	case meta.Stale:
		section.Summary = "We have not been able to check recently, so we cannot say which warnings are in force."
	default:
		section.Summary = "None in force as far as we know."
	}
	for _, bulletin := range in.Warnings.InForce {
		item := Item{ID: bulletin.ID, Title: bulletin.Title, Detail: bulletin.TypeLabel, Tone: bulletinTone(bulletin),
			Badge: bulletinBadge(bulletin), At: pointer(bulletin.IssuedAt), Stale: meta.Stale, Source: sourceBulletins,
			Target: &Target{Kind: "warnings", ID: bulletin.Type}, Href: pointer(bulletin.PDFURL)}
		if bulletin.MayBeWithdrawnBy != nil {
			item.Detail += ". It may already have been withdrawn; check with the DMC."
		}
		section.Items = append(section.Items, item)
	}
	return section
}

func bulletinBadge(bulletin warnings.Bulletin) string {
	word := "Advisory"
	if bulletin.Severity == string(dmcbulletins.SeverityWarning) {
		word = "Warning"
	}
	if bulletin.Colour != nil {
		return word + ": " + *bulletin.Colour
	}
	return word
}

func bulletinTone(bulletin warnings.Bulletin) string {
	if bulletin.Colour != nil {
		switch *bulletin.Colour {
		case "Red":
			return ToneSevere
		case "Amber":
			return ToneHigh
		case "Yellow":
			return ToneWatch
		}
	}
	if bulletin.Severity == string(dmcbulletins.SeverityWarning) {
		return ToneHigh
	}
	return ToneWatch
}

// gaugeReporting is the share of gauges that must be current for the gauges to
// speak for the country at all.
const gaugeReporting = 0.5

func gaugesSection(in Inputs) Section {
	into := &Target{Kind: "layer", Layer: "river-levels"}
	if in.Gauges == nil {
		return unavailable(SectionGauges, GroupWater, "River gauges", sourceGauges, false, false, into)
	}
	section := Section{ID: SectionGauges, Group: GroupWater, Title: "River gauges", Available: true,
		At: in.Gauges.Meta.FetchedAt, Source: sourceGauges, Counts: []Count{}, Items: []Item{}, Target: into,
		Note: "Measured at gauges, against the Irrigation Department's own levels. Most stretches of river have no gauge."}
	current, raised := 0, []rivers.StationProperties{}
	counts := map[rivers.Status]int{}
	for _, feature := range in.Gauges.Features {
		properties := feature.Properties
		if properties.Stale || properties.WaterLevel == nil {
			continue
		}
		current++
		counts[properties.Status]++
		if gaugeRank(properties.Status) > 0 {
			raised = append(raised, properties)
		}
	}
	total := len(in.Gauges.Features)
	section.Stale = total == 0 || float64(current) < gaugeReporting*float64(total)
	section.Counts = []Count{
		{Key: "major_flood", Label: "Major flood", Value: counts[rivers.StatusMajorFlood]},
		{Key: "minor_flood", Label: "Minor flood", Value: counts[rivers.StatusMinorFlood]},
		{Key: "alert", Label: "Alert", Value: counts[rivers.StatusAlert]},
		{Key: "reporting", Label: "Reporting", Value: current},
		{Key: "total", Label: "Gauges", Value: total},
	}
	sort.SliceStable(raised, func(i, j int) bool { return gaugeRank(raised[i].Status) > gaugeRank(raised[j].Status) })
	for _, properties := range raised {
		section.Items = append(section.Items, Item{ID: properties.ID, Title: properties.Name,
			Detail: gaugeDetail(properties), Tone: gaugeTone(properties.Status), Badge: gaugeLabels[properties.Status],
			At: properties.ObservedAt, Source: sourceGauges, Target: target("station", properties.ID)})
	}
	switch {
	case section.Stale:
		section.Summary = fmt.Sprintf("Only %d of %d gauges have a current reading, so we cannot say how the rivers stand.", current, total)
	case len(raised) == 0:
		section.Summary = fmt.Sprintf("None of the %d gauges reporting is at its alert level. %d not reporting.", current, total-current)
	default:
		section.Summary = fmt.Sprintf("%s at or above the alert level, of %d reporting.", plural(len(raised), "gauge", "gauges"), current)
	}
	return section
}

var gaugeLabels = map[rivers.Status]string{rivers.StatusAlert: "Alert", rivers.StatusMinorFlood: "Minor flood", rivers.StatusMajorFlood: "Major flood"}

func gaugeRank(status rivers.Status) int {
	switch status {
	case rivers.StatusMajorFlood:
		return 3
	case rivers.StatusMinorFlood:
		return 2
	case rivers.StatusAlert:
		return 1
	}
	return 0
}

func gaugeTone(status rivers.Status) string {
	switch status {
	case rivers.StatusMajorFlood:
		return ToneSevere
	case rivers.StatusMinorFlood:
		return ToneHigh
	}
	return ToneWatch
}

func gaugeDetail(properties rivers.StationProperties) string {
	reading := fmt.Sprintf("%.2f %s", *properties.WaterLevel, properties.Unit)
	if properties.Basin != "" {
		return properties.Basin + ", at " + reading
	}
	return reading
}

func riversSection(in Inputs) Section {
	into := &Target{Kind: "layer", Layer: "rivers"}
	if in.Risk == nil {
		return unavailable(SectionRivers, GroupWater, "Rivers at risk", sourceRisk, false, true, into)
	}
	section := Section{ID: SectionRivers, Group: GroupWater, Title: "Rivers at risk", Estimate: true, Available: true,
		At: in.Risk.Meta.GaugesObservedAt, Source: sourceRisk, Note: in.Risk.Note, Items: []Item{}, Target: into,
		Stale: in.Risk.Meta.GaugesCurrent == 0}
	counts := map[riverrisk.Level]int{}
	total := 0
	for _, river := range in.Risk.Rivers {
		for _, reach := range river.Reaches {
			counts[reach.Level]++
			total++
		}
	}
	section.Counts = []Count{
		{Key: "severe", Label: "Severe", Value: counts[riverrisk.LevelSevere]},
		{Key: "high", Label: "High", Value: counts[riverrisk.LevelHigh]},
		{Key: "watch", Label: "Watch", Value: counts[riverrisk.LevelWatch]},
		{Key: "normal", Label: "Normal", Value: counts[riverrisk.LevelNormal]},
		{Key: "unknown", Label: "Not known", Value: counts[riverrisk.LevelUnknown]},
	}
	atRisk := counts[riverrisk.LevelSevere] + counts[riverrisk.LevelHigh] + counts[riverrisk.LevelWatch]
	section.Summary = fmt.Sprintf("Estimate: %s of %d at watch or above. Nothing measures the %d not known.",
		plural(atRisk, "stretch", "stretches"), total, counts[riverrisk.LevelUnknown])
	rivers := append([]riverrisk.RiverRisk(nil), in.Risk.Rivers...)
	sort.SliceStable(rivers, func(i, j int) bool {
		if rivers[i].Level.Rank() != rivers[j].Level.Rank() {
			return rivers[i].Level.Rank() > rivers[j].Level.Rank()
		}
		return rivers[i].UplandKM > rivers[j].UplandKM
	})
	for _, river := range rivers {
		if !river.Level.AtRisk() || !river.Named || len(section.Items) == maxWorstRivers {
			continue
		}
		item := Item{ID: river.ID, Title: river.Name, Tone: riskTone(river.Level), Badge: riskLabels[river.Level], Estimate: true,
			At: firstTime(river.ObservedAt, in.Risk.Meta.GaugesObservedAt), Source: sourceRisk, Target: target("river", river.ID)}
		if len(river.Reasons) > 0 {
			item.Detail = river.Reasons[0].Text
		}
		for _, reach := range river.Reaches {
			if reach.Level == river.Level {
				item.Target.Reach = pointer(reach.FID)
				break
			}
		}
		section.Items = append(section.Items, item)
	}
	return section
}

var riskLabels = map[riverrisk.Level]string{riverrisk.LevelSevere: "Severe", riverrisk.LevelHigh: "High", riverrisk.LevelWatch: "Watch"}

func riskTone(level riverrisk.Level) string {
	switch level {
	case riverrisk.LevelSevere:
		return ToneSevere
	case riverrisk.LevelHigh:
		return ToneHigh
	case riverrisk.LevelWatch:
		return ToneWatch
	}
	return ToneNeutral
}

func reservoirsSection(in Inputs) Section {
	into := target("reservoirs", "")
	if in.Reservoirs == nil {
		section := unavailable(SectionReservoirs, GroupWater, "Reservoirs", sourceReservoirs, false, false, into)
		section.Note = reservoirNote
		return section
	}
	section := Section{ID: SectionReservoirs, Group: GroupWater, Title: "Reservoirs spilling or releasing", Available: true,
		At: in.Reservoirs.Meta.FetchedAt, Source: sourceReservoirs, Note: reservoirNote, Items: []Item{}, Target: into}
	spilling, releasing := []reservoirs.Properties{}, []reservoirs.Properties{}
	current := 0
	for _, feature := range in.Reservoirs.Features {
		properties := feature.Properties
		if properties.Stale {
			continue // a stale "spilling" does not count as spilling anywhere (ADR 0013)
		}
		current++
		switch {
		case properties.Status == "spilling":
			spilling = append(spilling, properties)
		case properties.Status == "releasing":
			releasing = append(releasing, properties)
		}
	}
	section.Stale = len(in.Reservoirs.Features) == 0 || current == 0
	section.Counts = []Count{
		{Key: "spilling", Label: "Spilling", Value: len(spilling)},
		{Key: "releasing", Label: "Releasing", Value: len(releasing)},
		{Key: "reporting", Label: "Current", Value: current},
		{Key: "total", Label: "Reservoirs", Value: len(in.Reservoirs.Features)},
	}
	for _, list := range [][]reservoirs.Properties{spilling, releasing} {
		sort.SliceStable(list, func(i, j int) bool { return flowFigure(list[i]) > flowFigure(list[j]) })
		for _, properties := range list {
			word, tone := "Spilling", ToneNeutral
			if figure := flowFigure(properties); figure >= spillNoticeCusecs || figure < 0 {
				tone = ToneWatch
			}
			if properties.Status == "releasing" {
				word, tone = "Releasing", ToneNeutral
			}
			section.Items = append(section.Items, Item{ID: properties.ID, Title: properties.Name, Detail: reservoirDetail(properties),
				Tone: tone, Badge: word, At: pointer(properties.FetchedAt), Source: sourceReservoirs, Target: target("reservoir", properties.ID)})
		}
	}
	switch {
	case section.Stale:
		section.Summary = "No current readings, so we cannot say which are spilling."
	case len(section.Items) == 0:
		section.Summary = fmt.Sprintf("None of the %d with a current reading is spilling or releasing.", current)
	default:
		section.Summary = fmt.Sprintf("%d spilling and %d releasing, of %d with a current reading.", len(spilling), len(releasing), current)
	}
	return section
}

func flowFigure(properties reservoirs.Properties) float64 {
	if properties.Flow != nil && properties.Flow.Cusecs != nil {
		return *properties.Flow.Cusecs
	}
	return -1
}

func reservoirDetail(properties reservoirs.Properties) string {
	var parts []string
	if flow := properties.Flow; flow != nil && flow.Cusecs != nil && !flow.Stale {
		switch flow.Kind {
		case "downstream":
			parts = append(parts, fmt.Sprintf("%s cusecs downstream, which can include power-station flow", thousands(*flow.Cusecs)))
		default:
			parts = append(parts, thousands(*flow.Cusecs)+" cusecs")
		}
	} else {
		parts = append(parts, "no figure given")
	}
	if properties.StoragePercent != nil && properties.StorageFresh != nil && *properties.StorageFresh {
		parts = append(parts, fmt.Sprintf("%.0f%% full", *properties.StoragePercent))
	}
	if properties.District != nil {
		parts = append(parts, *properties.District)
	}
	return strings.Join(parts, " · ")
}

func thousands(value float64) string {
	rounded := int(math.Round(value))
	if rounded >= 1000 {
		return fmt.Sprintf("%d,%03d", rounded/1000, rounded%1000)
	}
	if value < 10 && value != math.Trunc(value) {
		return fmt.Sprintf("%.1f", value)
	}
	return fmt.Sprint(rounded)
}

// Where a direction is measured from: the island's middle.
const (
	sriLankaLon = 80.77
	sriLankaLat = 7.87
)

// direction says which way a place lies from Sri Lanka, in eight words.
func direction(lon, lat float64) string {
	words := []string{"north", "north-east", "east", "south-east", "south", "south-west", "west", "north-west"}
	dx := (lon - sriLankaLon) * math.Cos(sriLankaLat*math.Pi/180)
	dy := lat - sriLankaLat
	bearing := math.Mod(math.Atan2(dx, dy)*180/math.Pi+360, 360)
	return words[int(math.Round(bearing/45))%8]
}

func cyclonesSection(in Inputs) Section {
	into := &Target{Kind: "layer", Layer: "cyclones"}
	if in.Cyclones == nil {
		section := unavailable(SectionCyclones, GroupHazards, "Cyclones", sourceCyclones, false, false, into)
		section.Note = "Check with the Department of Meteorology."
		return section
	}
	meta := in.Cyclones.Meta
	section := Section{ID: SectionCyclones, Group: GroupHazards, Title: "Cyclones in the North Indian Ocean", Available: true,
		Stale: meta.Stale, At: meta.FetchedAt, Source: sourceCyclones, Summary: meta.Note, Counts: []Count{}, Items: []Item{}, Target: into,
		Note: "GDACS's alert rates the impact on the countries it lists. It is not a warning for Sri Lanka."}
	for _, feature := range in.Cyclones.Features {
		properties := feature.Properties
		if properties["kind"] != "position" {
			continue
		}
		name, _ := properties["name"].(string)
		eventID, _ := properties["eventId"].(string)
		alert, _ := properties["alertLevel"].(string)
		lon, _ := properties["lon"].(float64)
		lat, _ := properties["lat"].(float64)
		distance, _ := properties["distanceKm"].(float64)
		stale, _ := properties["stale"].(bool)
		at, _ := properties["latestAt"].(time.Time)
		detail := fmt.Sprintf("%s km %s of Sri Lanka", thousands(distance), direction(lon, lat))
		if wind, ok := properties["windKmh"].(float64); ok {
			detail += fmt.Sprintf(" · winds to %.0f km/h", wind)
		}
		section.Items = append(section.Items, Item{ID: eventID, Title: name, Detail: detail, Tone: alertTone(alert),
			Badge: "GDACS " + alert, At: &at, Stale: stale || meta.Stale, Source: sourceCyclones, Target: target("cyclone", eventID)})
	}
	if len(section.Items) > 0 {
		section.Summary = plural(len(section.Items), "system", "systems") + " active."
	}
	return section
}

func alertTone(alert string) string {
	switch alert {
	case "Red":
		return ToneSevere
	case "Orange":
		return ToneHigh
	case "Green":
		return ToneWatch
	}
	return ToneNeutral
}

func earthquakesSection(in Inputs, now time.Time) Section {
	into := &Target{Kind: "layer", Layer: "earthquakes"}
	if in.Earthquakes == nil {
		return unavailable(SectionEarthquakes, GroupHazards, "Recent significant earthquakes", sourceQuakes, false, false, into)
	}
	meta := in.Earthquakes.Meta
	section := Section{ID: SectionEarthquakes, Group: GroupHazards, Title: "Recent significant earthquakes", Available: true,
		Stale: meta.Stale, At: meta.FetchedAt, Source: sourceQuakes, Counts: []Count{}, Items: []Item{}, Target: into,
		Note: "Lanka Weather generates no tsunami warnings. Follow the DMC."}
	for _, feature := range in.Earthquakes.Features {
		properties := feature.Properties
		if now.Sub(properties.Time) > significantDays*24*time.Hour || !significantQuake(properties) || len(section.Items) == maxQuakes {
			continue
		}
		detail := fmt.Sprintf("%s · %s km from Sri Lanka", properties.Place, thousands(properties.DistanceKm))
		tone := ToneNeutral
		if properties.Tsunami {
			detail += " · USGS tsunami flag, not a warning for Sri Lanka"
			tone = ToneWatch
		}
		section.Items = append(section.Items, Item{ID: properties.ID, Title: fmt.Sprintf("M%.1f earthquake", properties.Magnitude),
			Detail: detail, Tone: tone, At: pointer(properties.Time), Stale: meta.Stale, Source: sourceQuakes, Target: target("quake", properties.ID)})
	}
	switch {
	case len(section.Items) > 0:
		section.Summary = fmt.Sprintf("%s in the last %d days, of %d listed in %d.", plural(len(section.Items), "line", "lines"), significantDays, len(in.Earthquakes.Features), meta.WindowDays)
	case meta.Stale:
		section.Summary = "We have not been able to check USGS recently, so this may be out of date."
	default:
		section.Summary = fmt.Sprintf("None that stand out in the last %d days, of %d listed in %d.", significantDays, len(in.Earthquakes.Features), meta.WindowDays)
	}
	return section
}

func significantQuake(quake hazards.Earthquake) bool {
	return quake.Tsunami || quake.Magnitude >= quakeLargeM ||
		(quake.Magnitude >= quakeModerateM && quake.DistanceKm <= quakeModerateKm) || quake.DistanceKm <= quakeNearKm
}

func weatherSection(in Inputs) Section {
	into := &Target{Kind: "layer", Layer: "admin-regions", Fill: "temperature"}
	if in.Weather == nil {
		return unavailable(SectionWeather, GroupNow, "Weather by district", sourceWeather, false, true, into)
	}
	at := firstTime(in.Weather.ValidAt, in.Weather.FetchedAt)
	section := Section{ID: SectionWeather, Group: GroupNow, Title: "Weather by district", Estimate: true, Available: true,
		Stale: in.WeatherStale || len(in.Weather.Regions) == 0, At: at, Source: sourceWeather, Counts: []Count{}, Items: []Item{}, Target: into,
		Note: "Model values averaged over each district, not measurements."}
	line := func(id, title, name, regionID, detail, fill string) {
		section.Items = append(section.Items, Item{ID: id, Title: title + ": " + name, Detail: detail, Tone: ToneNeutral, Estimate: true,
			Stale: section.Stale, At: at, Source: sourceWeather, Target: &Target{Kind: "region", ID: regionID, Layer: "admin-regions", Fill: fill}})
	}
	byValue := func(value func(weather.RegionWeather) *float64) []weather.RegionWeather {
		out := make([]weather.RegionWeather, 0, len(in.Weather.Regions))
		for _, region := range in.Weather.Regions {
			if value(region) != nil {
				out = append(out, region)
			}
		}
		sort.SliceStable(out, func(i, j int) bool { return *value(out[i]) > *value(out[j]) })
		return out
	}
	temperature := func(region weather.RegionWeather) *float64 { return valueOf(region.TemperatureC) }
	if list := byValue(temperature); len(list) > 0 {
		hottest, coolest := list[0], list[len(list)-1]
		line("hottest", "Hottest", hottest.Name, hottest.ID, fmt.Sprintf("%.1f °C", *temperature(hottest)), "temperature")
		line("coolest", "Coolest", coolest.Name, coolest.ID, fmt.Sprintf("%.1f °C", *temperature(coolest)), "temperature")
	}
	rain := func(region weather.RegionWeather) *float64 { return valueOf(region.PrecipitationTodayMm) }
	for index, region := range byValue(rain) {
		if index == maxWettest || *rain(region) <= 0 {
			break
		}
		line(fmt.Sprintf("wettest-%d", index), "Wettest today", region.Name, region.ID, fmt.Sprintf("%.0f mm forecast today", *rain(region)), "rain")
	}
	wind := func(region weather.RegionWeather) *float64 { return valueOf(region.WindSpeedMaxKmh) }
	if list := byValue(wind); len(list) > 0 {
		strongest := list[0]
		detail := fmt.Sprintf("%.0f km/h", *wind(strongest))
		if gusts := valueOf(strongest.WindGustsMaxKmh); gusts != nil {
			detail += fmt.Sprintf(", gusts to %.0f km/h", *gusts)
		}
		line("windiest", "Strongest wind", strongest.Name, strongest.ID, detail, "wind")
	}
	section.Summary = "Hottest, coolest, wettest and windiest of the districts."
	if section.Stale {
		section.Summary = "These model values are out of date."
	}
	return section
}

func rainSection(in Inputs) Section {
	into := &Target{Kind: "layer", Layer: "rainfall"}
	if in.Rain == nil {
		return unavailable(SectionRain, GroupNow, "Rain at gauges, last 24 hours", sourceGauges, false, false, into)
	}
	section := Section{ID: SectionRain, Group: GroupNow, Title: "Rain at gauges, last 24 hours", Available: true,
		Source: sourceGauges, Counts: []Count{}, Items: []Item{}, Target: into, Note: "Measured at river gauges, which cluster in the wet zone."}
	fresh := []stations.RainProperties{}
	for _, feature := range in.Rain.Features {
		if feature.Properties.Kind == stations.KindGauge && !feature.Properties.Stale {
			fresh = append(fresh, feature.Properties)
		}
	}
	section.Stale = len(fresh) == 0
	sort.SliceStable(fresh, func(i, j int) bool { return fresh[i].RainMm > fresh[j].RainMm })
	for index, properties := range fresh {
		if index == maxRainGauges || properties.RainMm <= 0 {
			break
		}
		section.Items = append(section.Items, Item{ID: properties.ID, Title: properties.Name, Detail: fmt.Sprintf("%.1f mm", properties.RainMm), Tone: ToneNeutral,
			At: pointer(properties.WindowEnd), Source: sourceGauges, Target: &Target{Kind: "station", ID: properties.StationID, Layer: "rainfall"}})
	}
	if len(fresh) > 0 {
		section.At = pointer(fresh[0].WindowEnd)
	}
	switch {
	case section.Stale:
		section.Summary = "No gauge has enough current readings to give a total."
	case len(section.Items) == 0:
		section.Summary = fmt.Sprintf("No rain recorded at the %d gauges with a total.", len(fresh))
	default:
		section.Summary = fmt.Sprintf("The wettest of %d gauges with a total.", len(fresh))
	}
	return section
}

func airSection(in Inputs) Section {
	into := &Target{Kind: "layer", Layer: "air-quality"}
	if in.Air == nil {
		return unavailable(SectionAir, GroupNow, "Air quality", sourceAir, false, true, into)
	}
	place := in.Air.Place
	if place == "" {
		place = fmt.Sprintf("%.1f° N, %.1f° E", in.Air.Lat, in.Air.Lon)
	}
	category := aqiLabels[in.Air.Category]
	return Section{ID: SectionAir, Group: GroupNow, Title: "Air quality", Estimate: true, Available: true, Stale: in.Air.Stale,
		At: pointer(in.Air.FetchedAt), Source: sourceAir, Summary: "The worst value on the model's grid.", Note: "A model estimate about 40 km across. It does not measure the air at any place.",
		Counts: []Count{}, Target: into, Items: []Item{{ID: "worst", Title: "Worst: " + place, Detail: fmt.Sprintf("US AQI %.0f, %s", in.Air.USAQI, category),
			Tone: aqiTone(in.Air.USAQI), Estimate: true, Stale: in.Air.Stale, At: pointer(in.Air.FetchedAt), Source: sourceAir,
			Target: &Target{Kind: "place", Layer: "air-quality", Lon: pointer(in.Air.Lon), Lat: pointer(in.Air.Lat)}}}}
}

var aqiLabels = map[string]string{
	environment.AQIGood: "good", environment.AQIModerate: "moderate", environment.AQIUnhealthySensitive: "unhealthy for sensitive groups",
	environment.AQIUnhealthy: "unhealthy", environment.AQIVeryUnhealthy: "very unhealthy", environment.AQIHazardous: "hazardous",
}

func aqiTone(index float64) string {
	switch {
	case index > 200:
		return ToneSevere
	case index > 150:
		return ToneHigh
	case index > 100:
		return ToneWatch
	}
	return ToneNeutral
}

func seaSection(in Inputs) Section {
	into := &Target{Kind: "layer", Layer: "sea-conditions"}
	if in.Sea == nil {
		return unavailable(SectionSea, GroupNow, "Highest waves", sourceSea, false, true, into)
	}
	return Section{ID: SectionSea, Group: GroupNow, Title: "Highest waves", Estimate: true, Available: true, Stale: in.Sea.Stale,
		At: pointer(in.Sea.FetchedAt), Source: sourceSea, Summary: "The highest value on the model's grid near the coast.", Note: "Model values, not observations. Follow the Department of Meteorology's sea-area warnings.",
		Counts: []Count{}, Target: into, Items: []Item{{ID: "highest", Title: fmt.Sprintf("%.1f m, %s", in.Sea.WaveHeightM, in.Sea.State),
			Detail: fmt.Sprintf("About %.0f km %s of Sri Lanka's coast", in.Sea.DistanceKm, direction(in.Sea.Lon, in.Sea.Lat)), Tone: seaTone(in.Sea.WaveHeightM), Estimate: true,
			Stale: in.Sea.Stale, At: pointer(in.Sea.FetchedAt), Source: sourceSea, Target: &Target{Kind: "layer", Layer: "sea-conditions", Lon: pointer(in.Sea.Lon), Lat: pointer(in.Sea.Lat)}}}}
}

func seaTone(height float64) string {
	switch {
	case height >= 6:
		return ToneSevere
	case height >= 4:
		return ToneHigh
	case height >= 2.5:
		return ToneWatch
	}
	return ToneNeutral
}

func firstTime(times ...*time.Time) *time.Time {
	for _, at := range times {
		if at != nil {
			return at
		}
	}
	return nil
}

// valueOf is a region's value, or nil when it is missing.
func valueOf(value weather.Value) *float64 {
	number := float64(value)
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return nil
	}
	return &number
}
