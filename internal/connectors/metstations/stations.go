package metstations

import (
	"math"
	"strings"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
)

// Station is a Department of Meteorology station with a checked position.
type Station struct {
	ID         string
	Name       string
	WMOID      string
	Lat, Lon   float64
	ElevationM *float64
	// PositionSource says where the position comes from.
	PositionSource string
}

// Where positions come from.
const (
	// PositionISD is NOAA's Integrated Surface Database station history
	// (ncei.noaa.gov/pub/data/noaa/isd-history.txt), read 2026-11-02, public
	// domain, matched on the WMO number the 3-hourly source publishes.
	PositionISD = "noaa-isd"
	// PositionTable is a town position from the geocoded list, checked against
	// OpenStreetMap's place of that name (all within 2.1 km on 2026-11-02), for
	// stations that have no WMO number in the ISD list.
	PositionTable = "osm-checked"
	// PositionGeocoded is the 3-hourly source's sister list, geocoded by name;
	// used for rain-only stations and shown as approximate.
	PositionGeocoded = "nuuuwan-weather-lk-places"
)

func metres(value float64) *float64 { return &value }

// knownStations are the synoptic stations: the 24 of the 3-hourly report. All
// 24 report temperature in the daily report as well. Their numbers are the
// "Station ID" column of the source's README. Positions are never taken from
// the source's geocoded list for these, because it holds errors (see
// DecodePlaces); each position was compared with that list and they agree
// within 7 km, which a test holds.
var knownStations = []Station{
	{ID: "jaffna", Name: "Jaffna", WMOID: "43404", Lat: 9.650, Lon: 80.017, ElevationM: metres(3), PositionSource: PositionISD},
	{ID: "mullaitivu", Name: "Mullaitivu", WMOID: "43410", Lat: 9.267, Lon: 80.817, ElevationM: metres(2), PositionSource: PositionISD},
	{ID: "mannar", Name: "Mannar", WMOID: "43413", Lat: 8.983, Lon: 79.917, ElevationM: metres(3), PositionSource: PositionISD},
	{ID: "vavuniya", Name: "Vavuniya", WMOID: "43415", Lat: 8.750, Lon: 80.500, ElevationM: metres(98), PositionSource: PositionISD},
	{ID: "trincomalee", Name: "Trincomalee", WMOID: "43418", Lat: 8.539, Lon: 81.182, ElevationM: metres(1.8), PositionSource: PositionISD},
	{ID: "anuradhapura", Name: "Anuradhapura", WMOID: "43421", Lat: 8.301, Lon: 80.428, ElevationM: metres(98.8), PositionSource: PositionISD},
	{ID: "maha-illuppallama", Name: "Maha Illuppallama", WMOID: "43422", Lat: 8.117, Lon: 80.467, ElevationM: metres(117), PositionSource: PositionISD},
	{ID: "puttalam", Name: "Puttalam", WMOID: "43424", Lat: 8.033, Lon: 79.833, ElevationM: metres(2), PositionSource: PositionISD},
	{ID: "batticaloa", Name: "Batticaloa", WMOID: "43436", Lat: 7.706, Lon: 81.679, ElevationM: metres(6.1), PositionSource: PositionISD},
	{ID: "kurunegala", Name: "Kurunegala", WMOID: "43441", Lat: 7.467, Lon: 80.367, ElevationM: metres(116), PositionSource: PositionISD},
	{ID: "katugastota", Name: "Katugastota", WMOID: "43444", Lat: 7.333, Lon: 80.633, ElevationM: metres(477), PositionSource: PositionISD},
	{ID: "katunayake", Name: "Katunayake", WMOID: "43450", Lat: 7.181, Lon: 79.866, ElevationM: metres(8.8), PositionSource: PositionISD},
	{ID: "colombo", Name: "Colombo", WMOID: "43466", Lat: 6.900, Lon: 79.867, ElevationM: metres(7), PositionSource: PositionISD},
	{ID: "ratmalana", Name: "Ratmalana", WMOID: "43467", Lat: 6.822, Lon: 79.886, ElevationM: metres(6.7), PositionSource: PositionISD},
	{ID: "nuwara-eliya", Name: "Nuwara Eliya", WMOID: "43473", Lat: 6.967, Lon: 80.767, ElevationM: metres(1880), PositionSource: PositionISD},
	{ID: "pottuvil", Name: "Pottuvil", WMOID: "43475", Lat: 6.883, Lon: 81.833, ElevationM: metres(8), PositionSource: PositionISD},
	{ID: "bandarawela", Name: "Bandarawela", WMOID: "43476", Lat: 6.817, Lon: 80.967, ElevationM: metres(1250), PositionSource: PositionISD},
	{ID: "badulla", Name: "Badulla", WMOID: "43479", Lat: 6.983, Lon: 81.050, ElevationM: metres(667), PositionSource: PositionISD},
	{ID: "ratnapura", Name: "Ratnapura", WMOID: "43486", Lat: 6.683, Lon: 80.400, ElevationM: metres(88), PositionSource: PositionISD},
	{ID: "galle", Name: "Galle", WMOID: "43495", Lat: 6.033, Lon: 80.217, ElevationM: metres(13), PositionSource: PositionISD},
	{ID: "hambantota", Name: "Hambantota", WMOID: "43497", Lat: 6.117, Lon: 81.133, ElevationM: metres(20), PositionSource: PositionISD},
	// Local station numbers, not in the ISD list; see PositionTable.
	{ID: "mattala", Name: "Mattala", WMOID: "", Lat: 6.305, Lon: 81.114, PositionSource: PositionTable},
	{ID: "polonnaruwa", Name: "Polonnaruwa", WMOID: "", Lat: 7.940, Lon: 81.019, PositionSource: PositionTable},
	{ID: "moneragala", Name: "Moneragala", WMOID: "", Lat: 6.891, Lon: 81.345, PositionSource: PositionTable},
}

// aliases are other spellings of a known station in the two feeds.
var aliases = map[string]string{
	"mullativu":  "mullaitivu",
	"mulatiw":    "mullaitivu",
	"monaragala": "moneragala",
	// The 3-hourly source reported Katugastota as "Kandy" until December 2025.
	"kandy":            "katugastota",
	"mahailluppallama": "maha-illuppallama",
	"maha-illuppalama": "maha-illuppallama",
	"polonnaruva":      "polonnaruwa",
	"nuwaraeliya":      "nuwara-eliya",
	"barricaloa":       "batticaloa",
}

// ID is a station's identifier for a name as a feed writes it: lower case
// with hyphens, resolved through known spellings. "" for a name it cannot use.
func ID(name string) string {
	var builder strings.Builder
	hyphen := false
	for _, letter := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case letter >= 'a' && letter <= 'z' || letter >= '0' && letter <= '9':
			if hyphen && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			hyphen = false
			builder.WriteRune(letter)
		default:
			hyphen = true
		}
	}
	id := builder.String()
	if canonical, ok := aliases[id]; ok {
		return canonical
	}
	return id
}

// Known returns the synoptic stations.
func Known() []Station {
	return append([]Station(nil), knownStations...)
}

// KnownByID finds a synoptic station.
func KnownByID(id string) (Station, bool) {
	for _, station := range knownStations {
		if station.ID == id {
			return station, true
		}
	}
	return Station{}, false
}

// distanceKm is the great-circle distance between two positions.
func distanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0088
	toRadians := math.Pi / 180
	dLat, dLon := (lat2-lat1)*toRadians, (lon2-lon1)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// onSriLanka is whether a position is plausibly a station: on land, or within
// about 11 km of it, for coastal stations and small islands such as Mannar that
// the simplified coastline leaves out.
func onSriLanka(lat, lon float64) bool {
	const reach = 0.1
	for _, step := range [][2]float64{{0, 0}, {reach, 0}, {-reach, 0}, {0, reach}, {0, -reach}} {
		if grid.OnLand(lon+step[0], lat+step[1]) {
			return true
		}
	}
	return false
}
