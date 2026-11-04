// Package landslides collects NBRO's landslide early-warning levels per
// divisional secretariat division from nuuuwan/lk_dmc_landslides, which parses
// the reports the Disaster Management Centre publishes as PDF.
//
// data/all.json lists every report, newest first, as
//
//	{"date_id": "2026-10-31",
//	 "level_to_district_to_dsds": {"1": {"LK-21": ["LK-2136"]}, "2": {}, "3": {}}}
//
// The levels are NBRO's: 1 is yellow (watch), 2 amber (alert) and 3 red
// (evacuate). Codes are the same Department of Census and Statistics codes
// our regions use.
package landslides

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/regions"
)

// SourceName identifies the source in docs/data-sources.md.
const SourceName = "nuuuwan-lk-dmc-landslides"

// Attribution credits the issuer and the parser.
const Attribution = "National Building Research Organisation, Disaster Management Centre, via nuuuwan/lk_dmc_landslides"

// Level is one division's level in a report.
type Level struct {
	DS       string
	District string
	Level    int
}

// Report is one report, dated in Sri Lanka time.
type Report struct {
	// Date is midnight UTC of the report's local date.
	Date   time.Time
	Levels []Level
}

type wireReport struct {
	DateID string                                  `json:"date_id"`
	Levels map[string]map[string][]json.RawMessage `json:"level_to_district_to_dsds"`
}

// Decode reads data/all.json. Reports come back newest first. An entry that
// cannot be read is skipped and counted, so one odd entry does not lose the
// history; a file with no reports at all is an error.
func Decode(payload []byte) (reports []Report, skipped int, err error) {
	var wire []wireReport
	if err := json.Unmarshal(payload, &wire); err != nil {
		return nil, 0, fmt.Errorf("decode landslide reports: %w", err)
	}
	if len(wire) == 0 {
		return nil, 0, errors.New("no landslide reports")
	}
	seen := map[string]bool{}
	for _, entry := range wire {
		date, err := time.Parse(time.DateOnly, entry.DateID)
		if err != nil || seen[entry.DateID] {
			skipped++
			continue
		}
		seen[entry.DateID] = true
		report := Report{Date: date}
		// A division named at two levels takes the higher, the safer reading.
		best := map[string]Level{}
		for levelText, byDistrict := range entry.Levels {
			level, err := strconv.Atoi(levelText)
			if err != nil || level < 1 || level > 3 {
				skipped++
				continue
			}
			for _, list := range byDistrict {
				for _, raw := range list {
					var code string
					if err := json.Unmarshal(raw, &code); err != nil {
						skipped++
						continue
					}
					id, parsed, ok := regions.ParseID(code)
					if !ok || parsed != regions.LevelDS {
						skipped++
						continue
					}
					if current, ok := best[id]; !ok || level > current.Level {
						best[id] = Level{DS: id, District: regions.ParentID(id), Level: level}
					}
				}
			}
		}
		for _, level := range best {
			report.Levels = append(report.Levels, level)
		}
		sort.Slice(report.Levels, func(i, j int) bool { return report.Levels[i].DS < report.Levels[j].DS })
		reports = append(reports, report)
	}
	if len(reports) == 0 {
		return nil, skipped, errors.New("no readable landslide reports")
	}
	sort.SliceStable(reports, func(i, j int) bool { return reports[i].Date.After(reports[j].Date) })
	return reports, skipped, nil
}
