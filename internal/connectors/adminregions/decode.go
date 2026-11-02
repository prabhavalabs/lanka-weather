// Package adminregions collects Sri Lanka's administrative boundaries from
// nuuuwan/lk_admin_regions and their populations from nuuuwan/lk_census_2024.
//
// lk_admin_regions publishes every level as TopoJSON at several precisions.
// The collector reads the e4_medium files (positions quantised to a few
// metres, simplified to about 10 m), whose properties carry each region's
// DCS code, English name, Sinhala and Tamil names, area and centre.
// lk_census_2024 publishes the 2024 Census of Population and Housing per
// region with the same codes.
package adminregions

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/geo"
	"github.com/prabhavalabs/lanka-weather/internal/names"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
)

// Region is one decoded administrative region.
type Region struct {
	ID         string
	FID        int64
	Level      regions.Level
	Name       string
	NameSI     string
	NameTA     string
	ParentID   string
	ProvinceID string
	DistrictID string
	DSID       string
	ISOCode    string
	GNNumber   string
	// AreaKM2 is the published area; zero when the source has none.
	AreaKM2              float64
	CenterLon, CenterLat float64
	Shape, Simple        geo.MultiPolygon
	// Population is the census count, when the census lists the region.
	Population *Population
}

type properties struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	AreaSqKm   *float64 `json:"area_sqkm"`
	CenterLat  *float64 `json:"center_lat"`
	CenterLng  *float64 `json:"center_lng"`
	OtherNames *string  `json:"other_names"`
	Num        *string  `json:"num"`
}

// Sri Lanka with a margin.
const minLon, maxLon, minLat, maxLat = 79.0, 82.5, 5.5, 10.5

// DecodeLevel decodes one level's TopoJSON file. Features without a DCS code
// of that level (the source keeps a few unmatched shapes) are skipped and
// counted. It does no I/O.
func DecodeLevel(level regions.Level, payload []byte) ([]Region, int, error) {
	topology, err := geo.DecodeTopology(payload)
	if err != nil {
		return nil, 0, err
	}
	objects := topology.Objects()
	if len(objects) != 1 {
		return nil, 0, fmt.Errorf("%s topology has %d objects, want 1", level, len(objects))
	}
	features, skipped, err := topology.Features(objects[0], regions.Tolerances[level])
	if err != nil {
		return nil, 0, fmt.Errorf("%s topology: %w", level, err)
	}
	out := make([]Region, 0, len(features))
	seen := make(map[string]bool, len(features))
	for _, feature := range features {
		var props properties
		if len(feature.Properties) > 0 && !bytes.Equal(feature.Properties, []byte("null")) {
			if err := json.Unmarshal(feature.Properties, &props); err != nil {
				return nil, 0, fmt.Errorf("%s properties: %w", level, err)
			}
		}
		id, got, ok := regions.ParseID(props.ID)
		name := strings.TrimSpace(props.Name)
		if !ok || got != level || name == "" || seen[id] {
			skipped++
			continue
		}
		seen[id] = true
		fid, _ := regions.FID(id)
		province, district, ds := regions.Ancestry(id)
		region := Region{
			ID: id, FID: fid, Level: level, Name: name,
			ParentID: regions.ParentID(id), ProvinceID: province, DistrictID: district, DSID: ds,
			ISOCode: regions.ISOCode(id),
			Shape:   feature.Shape, Simple: feature.Simple,
		}
		if props.OtherNames != nil {
			region.NameSI, region.NameTA = SplitOtherNames(*props.OtherNames)
		}
		if level == regions.LevelGN && props.Num != nil {
			region.GNNumber = strings.TrimSpace(*props.Num)
		}
		if props.AreaSqKm != nil && *props.AreaSqKm > 0 && !math.IsInf(*props.AreaSqKm, 0) {
			region.AreaKM2 = *props.AreaSqKm
		}
		region.CenterLon, region.CenterLat = center(props, feature.Shape)
		out = append(out, region)
	}
	return out, skipped, nil
}

// center returns the published centre, or the middle of the shape's box when
// the published one is missing or outside Sri Lanka.
func center(props properties, shape geo.MultiPolygon) (float64, float64) {
	if props.CenterLng != nil && props.CenterLat != nil {
		lon, lat := *props.CenterLng, *props.CenterLat
		if lon >= minLon && lon <= maxLon && lat >= minLat && lat <= maxLat {
			return lon, lat
		}
	}
	box := shape.Bounds()
	return (box.West + box.East) / 2, (box.South + box.North) / 2
}

// SplitOtherNames separates the source's comma-separated other names into
// Sinhala and Tamil by script. Names may themselves contain commas, so the
// parts in each script are joined again; parts in Latin script (overflow of
// an English name containing a comma) are ignored.
func SplitOtherNames(raw string) (sinhala, tamil string) {
	var si, ta []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		switch names.Of(part) {
		case names.ScriptSinhala:
			si = append(si, part)
		case names.ScriptTamil:
			ta = append(ta, part)
		}
	}
	return strings.Join(si, ", "), strings.Join(ta, ", ")
}

// Population is a region's census population.
type Population struct {
	Total int
	Year  int
}

// CensusYear is the census lk_census_2024 publishes.
const CensusYear = 2024

// DecodeCensus reads the census table of population by sex (data.tsv), whose
// columns include region_id, region_ent_type and total_value, and returns
// the total population of every province, district, DS and GN division.
func DecodeCensus(payload []byte) (map[string]Population, error) {
	reader := csv.NewReader(bytes.NewReader(payload))
	reader.Comma = '\t'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read census header: %w", err)
	}
	column := map[string]int{}
	for index, name := range header {
		column[strings.TrimSpace(name)] = index
	}
	for _, required := range []string{"region_id", "region_ent_type", "total_value"} {
		if _, ok := column[required]; !ok {
			return nil, fmt.Errorf("census table has no %s column", required)
		}
	}
	out := map[string]Population{}
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read census row: %w", err)
		}
		field := func(name string) string {
			if index := column[name]; index < len(record) {
				return strings.TrimSpace(record[index])
			}
			return ""
		}
		switch field("region_ent_type") {
		case "province", "district", "dsd", "gnd":
		default:
			continue
		}
		id, _, ok := regions.ParseID(field("region_id"))
		total, err := strconv.Atoi(field("total_value"))
		if !ok || err != nil || total < 0 {
			continue
		}
		out[id] = Population{Total: total, Year: CensusYear}
	}
	if len(out) == 0 {
		return nil, errors.New("census table has no regions")
	}
	return out, nil
}

// Commit is the latest commit that changed a path of a repository.
type Commit struct {
	SHA  string
	Time time.Time
}

// DecodeCommits reads the GitHub API's list of commits for a path and returns
// the newest.
func DecodeCommits(payload []byte) (Commit, error) {
	var commits []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date time.Time `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(payload, &commits); err != nil {
		return Commit{}, fmt.Errorf("decode commits: %w", err)
	}
	if len(commits) == 0 || len(commits[0].SHA) != 40 || commits[0].Commit.Committer.Date.IsZero() {
		return Commit{}, errors.New("decode commits: no commit with a SHA and date")
	}
	return Commit{SHA: commits[0].SHA, Time: commits[0].Commit.Committer.Date.UTC()}, nil
}
