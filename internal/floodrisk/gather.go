package floodrisk

import (
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/googleflood"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/waterlevels"
	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/raingauges"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
	"github.com/prabhavalabs/lanka-weather/internal/weather"
)

// RainFreshFor is how old a gauge's newest reading may be to count.
const RainFreshFor = 90 * time.Minute

// Layers are what the sources serve now. A nil layer was not read: its
// families say nothing, which never lowers the estimate to calm on its own
// (the engine treats a run missing a layer as incomplete).
type Layers struct {
	Rain       *raingauges.Intensities
	Floods     *floods.FeatureCollection
	Rivers     *rivers.FeatureCollection
	Bulletins  *warnings.Overview
	Reservoirs *reservoirs.FeatureCollection
	// Outlook is the forecast rain ahead by district.
	Outlook *weather.RainOutlooks
}

// Complete reports whether every layer was read.
func (layers Layers) Complete() bool {
	return layers.Rain != nil && layers.Floods != nil && layers.Rivers != nil && layers.Bulletins != nil &&
		layers.Reservoirs != nil && layers.Outlook != nil
}

// Places finds districts.
type Places struct {
	DistrictOf    func(lon, lat float64) string
	AreaDistricts func(areaID string) []string
	DistrictNamed func(name string) string
}

// floodBulletinTypes are the bulletins that speak of rain or rivers.
var floodBulletinTypes = []string{"weather", "water_level", "situation"}

// officialLevel is a bulletin's level: red 3, warning or amber 2, advisory 1.
func officialLevel(bulletin warnings.Bulletin) int {
	colour := ""
	if bulletin.Colour != nil {
		colour = strings.ToLower(strings.TrimSpace(*bulletin.Colour))
	}
	switch {
	case colour == "red":
		return 3
	case bulletin.Severity == "warning", colour == "amber":
		return 2
	case bulletin.Severity == "advisory":
		return 1
	}
	return 0
}

func riverLevel(status rivers.Status) int {
	switch status {
	case rivers.StatusAlert:
		return 1
	case rivers.StatusMinorFlood:
		return 2
	case rivers.StatusMajorFlood:
		return 3
	}
	return 0
}

// Gather builds each district's inputs from the layers. Every district with a
// forecast is present, so a district with nothing else still has an entry.
func Gather(layers Layers, places Places, now time.Time) map[string]Inputs {
	inputs := map[string]*Inputs{}
	get := func(district string) *Inputs {
		if entry, ok := inputs[district]; ok {
			return entry
		}
		entry := &Inputs{District: district}
		inputs[district] = entry
		return entry
	}
	if outlook := layers.Outlook; outlook != nil && !outlook.Stale {
		for district, rain := range outlook.Regions {
			get(district).Forecast = &ForecastRain{Next6hMaxMm: rain.Next6hMaxMm, Next24hMeanMm: rain.Next24hMeanMm}
		}
	}
	if rain := layers.Rain; rain != nil {
		for _, gauge := range rain.Gauges {
			if gauge.Suspect || gauge.Stale || now.Sub(gauge.ObservedAt) > RainFreshFor {
				continue
			}
			district := places.DistrictOf(gauge.Lon, gauge.Lat)
			if district == "" {
				continue
			}
			entry := get(district)
			if entry.Rain == nil {
				entry.Rain = &ObservedRain{}
			}
			observed := entry.Rain
			// The gauge named is the one with the largest three-hour rise,
			// which holds any burst of the last hour too.
			if observed.GaugeID == "" || gauge.Rise3hMm > observed.Rise3hMm {
				observed.Gauge, observed.GaugeID = gauge.Name, gauge.ID
			}
			observed.Rise1hMm = max(observed.Rise1hMm, gauge.Rise1hMm)
			observed.Rise3hMm = max(observed.Rise3hMm, gauge.Rise3hMm)
			observed.PriorMm = max(observed.PriorMm, round1(max(0, gauge.Rain24hMm-gauge.Rise3hMm)))
		}
	}
	if layer := layers.Floods; layer != nil && !layer.Meta.Seeded {
		for _, feature := range layer.Features {
			switch properties := feature.Properties.(type) {
			case floods.Gauge:
				if !properties.QualityVerified || properties.Stale || properties.Seeded {
					continue
				}
				level := googleflood.Severity(properties.Severity).Rank() - 1
				if level <= 0 {
					continue
				}
				if district := places.DistrictOf(properties.Lon, properties.Lat); district != "" {
					entry := get(district)
					entry.GaugeForecastLevel = max(entry.GaugeForecastLevel, level)
				}
			case floods.Area:
				level := 0
				switch properties.Kind {
				case googleflood.KindFlashLikely:
					level = 1
				case googleflood.KindFlashHighly:
					level = 2
				}
				if level == 0 {
					continue
				}
				for _, district := range places.AreaDistricts(properties.ID) {
					entry := get(district)
					entry.FlashLevel = max(entry.FlashLevel, level)
				}
			}
		}
	}
	if layer := layers.Rivers; layer != nil {
		for _, feature := range layer.Features {
			station := feature.Properties
			if station.Source == floods.SeedSource || station.Stale {
				continue
			}
			level := riverLevel(station.Status)
			if level == 0 {
				continue
			}
			if district := places.DistrictOf(feature.Geometry.Coordinates[0], feature.Geometry.Coordinates[1]); district != "" {
				entry := get(district)
				entry.RiverLevel = max(entry.RiverLevel, level)
			}
		}
	}
	if overview := layers.Bulletins; overview != nil && !overview.Meta.Stale {
		for _, bulletin := range overview.InForce {
			if !contains(floodBulletinTypes, bulletin.Type) {
				continue
			}
			level := officialLevel(bulletin)
			if level == 0 {
				continue
			}
			// A bulletin naming no district is about the whole island and too
			// broad to raise any one district.
			for _, district := range bulletin.Districts {
				entry := get(district.ID)
				entry.OfficialLevel = max(entry.OfficialLevel, level)
			}
		}
	}
	if layer := layers.Reservoirs; layer != nil {
		for _, feature := range layer.Features {
			properties := feature.Properties
			if properties.Stale || properties.District == nil || properties.Status != waterlevels.StatusSpilling {
				continue
			}
			if district := places.DistrictNamed(*properties.District); district != "" {
				get(district).Spilling++
			}
		}
	}
	out := make(map[string]Inputs, len(inputs))
	for district, entry := range inputs {
		out[district] = *entry
	}
	return out
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func round1(value float64) float64 { return float64(int(value*10+0.5)) / 10 }
