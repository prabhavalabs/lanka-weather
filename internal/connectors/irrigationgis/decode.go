// Package irrigationgis reads the Irrigation Department's public ArcGIS
// services other than its gauge readings: the official river basins, named
// streams and station list, which give the map its country-level boundaries
// and names, and the mapped extents of past floods.
//
// The services publish no licence or terms (ADR 0010); the department is
// credited wherever the data is shown.
package irrigationgis

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/geo"
)

// Source names, as listed in docs/data-sources.md.
const (
	BasinsSource     = "irrigation-river-basins"
	PastFloodsSource = "irrigation-past-floods"
)

// Attribution is the credit shown with the department's data.
const Attribution = "Irrigation Department of Sri Lanka"

// Feature is one record of a feature service page.
type Feature struct {
	Attributes map[string]any
	Rings      []geo.Ring
}

// Text reads a text attribute, trimmed; anything else is empty.
func (feature Feature) Text(name string) string {
	value, _ := feature.Attributes[name].(string)
	return strings.TrimSpace(value)
}

// Number reads a numeric attribute.
func (feature Feature) Number(name string) (float64, bool) {
	value, ok := feature.Attributes[name].(float64)
	return value, ok
}

// Shape is the feature's polygon.
func (feature Feature) Shape() geo.MultiPolygon { return geo.PolygonFromRings(feature.Rings) }

// DecodePage reads one page of a query. more reports that the service holds
// further records. It does no I/O.
func DecodePage(payload []byte) (features []Feature, more bool, err error) {
	var page struct {
		Features []struct {
			Attributes map[string]any `json:"attributes"`
			Geometry   *struct {
				Rings [][]geo.Point `json:"rings"`
			} `json:"geometry"`
		} `json:"features"`
		Exceeded bool `json:"exceededTransferLimit"`
		Error    *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &page); err != nil {
		return nil, false, fmt.Errorf("decode feature page: %w", err)
	}
	if page.Error != nil {
		return nil, false, fmt.Errorf("feature service error %d: %s", page.Error.Code, page.Error.Message)
	}
	if page.Features == nil {
		return nil, false, errors.New("decode feature page: no features")
	}
	features = make([]Feature, 0, len(page.Features))
	for _, wire := range page.Features {
		feature := Feature{Attributes: wire.Attributes}
		if wire.Geometry != nil {
			feature.Rings = make([]geo.Ring, len(wire.Geometry.Rings))
			for index, ring := range wire.Geometry.Rings {
				feature.Rings[index] = geo.Ring(ring)
			}
		}
		features = append(features, feature)
	}
	return features, page.Exceeded, nil
}

// LayerInfo is what a layer's description says about it.
type LayerInfo struct {
	ID       int
	Name     string
	LastEdit int64
}

// DecodeServiceLayers reads the layers of a service's description.
func DecodeServiceLayers(payload []byte) ([]LayerInfo, error) {
	var service struct {
		Layers []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(payload, &service); err != nil {
		return nil, fmt.Errorf("decode service: %w", err)
	}
	if len(service.Layers) == 0 {
		return nil, errors.New("decode service: no layers")
	}
	layers := make([]LayerInfo, len(service.Layers))
	for index, layer := range service.Layers {
		layers[index] = LayerInfo{ID: layer.ID, Name: strings.TrimSpace(layer.Name)}
	}
	return layers, nil
}

// DecodeLastEdit reads when a layer was last edited (milliseconds since 1970),
// or 0 if the layer does not say.
func DecodeLastEdit(payload []byte) int64 {
	var layer struct {
		EditingInfo struct {
			LastEditDate int64 `json:"lastEditDate"`
		} `json:"editingInfo"`
	}
	_ = json.Unmarshal(payload, &layer)
	return layer.EditingInfo.LastEditDate
}

// Basin is one cleaned official river basin.
type Basin struct {
	FID   int64
	ID    string
	Name  string
	Named bool
	Shape geo.MultiPolygon
}

// basinPlaceholder is what the service calls a polygon it has not named.
const basinPlaceholder = "Placemark"

// CoastalName is the name of an unnamed polygon: a stretch of coast draining
// directly to the sea, outside any named basin.
const CoastalName = "Coastal area"

// minBasinKM2 is the smallest polygon kept on its own. Below it lie the 511
// unnamed slivers along basin edges and coasts, which are converted drawing
// artefacts and together cover less than 100 km², and the 12 named scraps.
const minBasinKM2 = 1.0

// basinNameFixes corrects spelling mistakes in the source.
var basinNameFixes = map[string]string{"Telwatta Ganaga": "Telwatta Ganga"}

// areaKM2 approximates a shape's area, enough to tell a sliver from a basin.
func areaKM2(shape geo.MultiPolygon) float64 {
	total := 0.0
	for _, polygon := range shape {
		for index, ring := range polygon {
			if len(ring) < 4 {
				continue
			}
			scale := 111.32 * 111.32 * math.Cos(ring[0][1]*math.Pi/180)
			area := 0.0
			for i := 1; i < len(ring); i++ {
				area += ring[i-1][0]*ring[i][1] - ring[i][0]*ring[i-1][1]
			}
			area = math.Abs(area) / 2 * scale
			if index > 0 {
				area = -area
			}
			total += area
		}
	}
	return total
}

// Slug makes an identifier of a name: "Kala Oya" is "kala-oya".
func Slug(name string) string {
	var out strings.Builder
	dash := false
	for _, letter := range strings.ToLower(name) {
		if letter >= 'a' && letter <= 'z' || letter >= '0' && letter <= '9' {
			if dash && out.Len() > 0 {
				out.WriteByte('-')
			}
			out.WriteRune(letter)
			dash = false
		} else {
			dash = true
		}
	}
	return out.String()
}

// CleanBasins turns the service's 727 polygons into usable basins:
//
//   - a named scrap is merged with the largest polygon of the same name;
//   - an unnamed sliver is dropped;
//   - an unnamed polygon that is large enough is kept as a coastal area;
//   - two large polygons sharing a name stay apart: Pali Aru names two
//     different rivers 50 km from each other.
//
// Pure, so it is tested without a network.
func CleanBasins(features []Feature) []Basin {
	type piece struct {
		fid   int64
		name  string
		shape geo.MultiPolygon
		area  float64
	}
	var pieces []piece
	largest := map[string]int{}
	for _, feature := range features {
		fid, ok := feature.Number("FID")
		if !ok {
			continue
		}
		name := feature.Text("Name")
		if name == basinPlaceholder {
			name = ""
		}
		if fixed, ok := basinNameFixes[name]; ok {
			name = fixed
		}
		shape := feature.Shape()
		if len(shape) == 0 {
			continue
		}
		pieces = append(pieces, piece{fid: int64(fid), name: name, shape: shape, area: areaKM2(shape)})
		index := len(pieces) - 1
		if name != "" {
			if best, ok := largest[name]; !ok || pieces[best].area < pieces[index].area {
				largest[name] = index
			}
		}
	}
	merged := map[int]geo.MultiPolygon{}
	var kept []int
	for index, candidate := range pieces {
		switch {
		case candidate.area >= minBasinKM2:
			kept = append(kept, index)
		case candidate.name != "":
			// A scrap of a named basin belongs with it, unless the whole basin
			// is a scrap, in which case it is lost with the other slivers.
			best := largest[candidate.name]
			if best != index && pieces[best].area >= minBasinKM2 {
				merged[best] = append(merged[best], candidate.shape...)
			}
		}
	}
	basins := make([]Basin, 0, len(kept))
	used := map[string]bool{}
	sort.SliceStable(kept, func(i, j int) bool { return pieces[kept[i]].area > pieces[kept[j]].area })
	for _, index := range kept {
		candidate := pieces[index]
		basin := Basin{FID: candidate.fid, Named: candidate.name != "", Name: candidate.name,
			Shape: append(candidate.shape, merged[index]...)}
		if basin.Named {
			basin.ID = Slug(candidate.name)
			if used[basin.ID] {
				basin.ID += "-" + strconv.FormatInt(candidate.fid, 10)
			}
		} else {
			basin.Name = CoastalName
			basin.ID = "coastal-" + strconv.FormatInt(candidate.fid, 10)
		}
		used[basin.ID] = true
		basins = append(basins, basin)
	}
	return basins
}

// Stream is a named stream polygon of the department's map.
type Stream struct {
	FID   int64
	Name  string
	Shape geo.MultiPolygon
}

// streamNameNoise lists values of NAME that do not name a river.
var streamNameNoise = map[string]bool{"flood outlet": true, "spill": true, "ela": true, "canal": true}

// titleCase writes "MAHAWELI GANGA" as "Mahaweli Ganga"; names already in
// mixed case keep it.
func titleCase(name string) string {
	if name != strings.ToUpper(name) {
		return name
	}
	words := strings.Fields(strings.ToLower(name))
	for index, word := range words {
		words[index] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

// DecodeStreams keeps the named stream polygons.
func DecodeStreams(features []Feature) []Stream {
	var streams []Stream
	for _, feature := range features {
		fid, ok := feature.Number("FID")
		name := titleCase(feature.Text("NAME"))
		shape := feature.Shape()
		if !ok || name == "" || streamNameNoise[strings.ToLower(name)] || len(shape) == 0 {
			continue
		}
		streams = append(streams, Stream{FID: int64(fid), Name: name, Shape: shape})
	}
	return streams
}

// Station is one row of the department's station list.
type Station struct {
	Name, Basin, Tributary string
	Lon, Lat               float64
}

// DecodeStations reads the station list. Stations without a position are
// skipped.
func DecodeStations(features []Feature) []Station {
	var stations []Station
	for _, feature := range features {
		lat, latOK := feature.Number("latitude")
		lon, lonOK := feature.Number("longitude")
		name := feature.Text("station")
		if !latOK || !lonOK || name == "" {
			continue
		}
		stations = append(stations, Station{Name: name, Basin: feature.Text("basin"), Tributary: feature.Text("Tributory"), Lon: lon, Lat: lat})
	}
	return stations
}

// PastEvent is one mapped past flood.
type PastEvent struct {
	FID int64
	ID  string
	// Name is for reading: "Kelani Ganga flood, 2016 (major)".
	Name string
	// SourceName is the value as the service writes it, spelling included.
	SourceName string
	River      string
	Year       int
	// Date is set only when the service names a day.
	Date      *time.Time
	Magnitude string
	Service   string
	Shape     geo.MultiPolygon
}

var (
	yearPattern      = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	magnitudePattern = regexp.MustCompile(`(?i)\b(critical|major|minor)\b`)
	// Words that follow the river in the source's event names.
	floodWords = regexp.MustCompile(`(?i)\b(flood|inundation|nov|critical|major|minor)\b`)
	// A basin number in brackets: "Maha Oya (102)".
	basinNumber = regexp.MustCompile(`\s*\(\d+\)`)
)

// riverSpellings corrects river names as the flood services write them.
var riverSpellings = map[string]string{
	"kelani ganag":   "Kelani Ganga",
	"mundani aru":    "Mundeni Aru",
	"attanagala oya": "Attanagalu Oya",
	"mahawali":       "Mahaweli Ganga",
	"mahaweli":       "Mahaweli Ganga",
}

// ParseEvent reads an event's name, as in "Kelani Ganag Flood 2016 Major" or
// "Attanagala Oya Flood Inundation 2025 Nov". It reports false when the value
// has no year: a past event must say when it was. Flood_Map's layers must be
// matched by this value, not their ids, which do not line up across its
// repeated layer groups.
func ParseEvent(source string) (PastEvent, bool) {
	source = strings.TrimSpace(source)
	match := yearPattern.FindString(source)
	if match == "" {
		return PastEvent{}, false
	}
	year, _ := strconv.Atoi(match)
	magnitude := strings.ToLower(magnitudePattern.FindString(source))
	river := yearPattern.ReplaceAllString(source, " ")
	river = floodWords.ReplaceAllString(river, " ")
	river = basinNumber.ReplaceAllString(river, "")
	river = strings.Join(strings.Fields(river), " ")
	if fixed, ok := riverSpellings[strings.ToLower(river)]; ok {
		river = fixed
	}
	if river == "" {
		return PastEvent{}, false
	}
	name := fmt.Sprintf("%s flood, %d", river, year)
	if magnitude != "" {
		name += " (" + magnitude + ")"
	}
	event := PastEvent{SourceName: source, River: river, Year: year, Magnitude: magnitude, Name: name}
	event.identify()
	return event, true
}

func (event *PastEvent) identify() {
	event.ID = Slug(event.Name)
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(event.ID))
	// Vector tile feature ids; 32 bits are exact in JavaScript.
	event.FID = int64(hash.Sum32())
}

// PuttalamEvent describes the one event the Puttalam service holds. It says
// so only in its name (which spells the cyclone "Detwah"), and each of its
// 31,985 features is a single raster cell.
func PuttalamEvent() PastEvent {
	date := time.Date(2025, 11, 30, 0, 0, 0, 0, time.UTC)
	event := PastEvent{SourceName: "Puttalam_Flood__Detwah_30Nov2025", River: "Puttalam area", Year: 2025, Date: &date,
		Name: "Puttalam flood, Cyclone Ditwah, 30 Nov 2025"}
	event.identify()
	return event
}
