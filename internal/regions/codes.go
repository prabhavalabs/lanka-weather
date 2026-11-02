// Package regions serves Sri Lanka's administrative regions: boundaries as
// GeoJSON and vector tiles, region details, and point lookup.
//
// A region is identified by its Department of Census and Statistics (DCS)
// code written with one hyphen after the country prefix, as
// nuuuwan/lk_admin_regions publishes it:
//
//	province  LK-1          one digit
//	district  LK-11         two digits
//	ds        LK-1127       four digits
//	gn        LK-1127005    seven digits
//
// A region's code starts with its parent's code. Other spellings of the same
// code (LK11 as used by lanka-data-layer, LK-1-1-27) are accepted and
// normalised. ISO 3166-2 codes equal the DCS codes for provinces and for every
// district except the Northern Province's; ISOCode maps them.
package regions

import (
	"strconv"
	"strings"
)

// Level is an administrative level.
type Level string

const (
	LevelProvince Level = "province"
	LevelDistrict Level = "district"
	// LevelDS is the divisional secretariat division.
	LevelDS Level = "ds"
	// LevelGN is the Grama Niladhari division.
	LevelGN Level = "gn"
)

// Levels lists the levels from largest to smallest.
var Levels = []Level{LevelProvince, LevelDistrict, LevelDS, LevelGN}

// digits is the number of code digits at each level.
var digits = map[Level]int{LevelProvince: 1, LevelDistrict: 2, LevelDS: 4, LevelGN: 7}

// ParseLevel returns the level named s.
func ParseLevel(s string) (Level, bool) {
	level := Level(s)
	_, ok := digits[level]
	return level, ok
}

// Rank is the level's depth: 1 for a province to 4 for a GN division.
func (level Level) Rank() int {
	for index, candidate := range Levels {
		if candidate == level {
			return index + 1
		}
	}
	return 0
}

// ParseID normalises a region code and returns its level. It accepts the
// canonical form (LK-1127), the compact form (LK1127), lower case and the
// hyphenated form (LK-1-1-27).
func ParseID(raw string) (string, Level, bool) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if !strings.HasPrefix(code, "LK") {
		return "", "", false
	}
	number := strings.ReplaceAll(code[2:], "-", "")
	if number == "" {
		return "", "", false
	}
	for _, r := range number {
		if r < '0' || r > '9' {
			return "", "", false
		}
	}
	for _, level := range Levels {
		if len(number) == digits[level] {
			return "LK-" + number, level, true
		}
	}
	return "", "", false
}

// ParentID returns the code of the region containing id, or "" for a
// province or an invalid code.
func ParentID(id string) string {
	canonical, level, ok := ParseID(id)
	if !ok || level == LevelProvince {
		return ""
	}
	parent := Levels[level.Rank()-2]
	return canonical[:len("LK-")+digits[parent]]
}

// Ancestry returns the province, district and DS division containing id,
// counting the region itself at its own level; levels below it are "".
func Ancestry(id string) (province, district, ds string) {
	canonical, level, ok := ParseID(id)
	if !ok {
		return "", "", ""
	}
	number := canonical[3:]
	at := func(l Level) string {
		if level.Rank() < l.Rank() {
			return ""
		}
		return "LK-" + number[:digits[l]]
	}
	return at(LevelProvince), at(LevelDistrict), at(LevelDS)
}

// FID is the code's digits as a number. Codes at different levels have
// different numbers of digits, so FIDs are unique across levels; vector tiles
// use them as feature ids.
func FID(id string) (int64, bool) {
	canonical, _, ok := ParseID(id)
	if !ok {
		return 0, false
	}
	value, err := strconv.ParseInt(canonical[3:], 10, 64)
	return value, err == nil
}

// isoDistricts holds the districts whose ISO 3166-2 code differs from their
// DCS code: the Northern Province's districts are numbered in a different
// order.
var isoDistricts = map[string]string{
	"LK-42": "LK-43", // Mannar
	"LK-43": "LK-44", // Vavuniya
	"LK-44": "LK-45", // Mullaitivu
	"LK-45": "LK-42", // Kilinochchi
}

// ISOCode returns the ISO 3166-2 code of a province or district, or "" for
// other levels.
func ISOCode(id string) string {
	canonical, level, ok := ParseID(id)
	if !ok || (level != LevelProvince && level != LevelDistrict) {
		return ""
	}
	if iso, differs := isoDistricts[canonical]; differs {
		return iso
	}
	return canonical
}

// Tolerances are the simplification tolerances, in degrees, of each level's
// simplified boundary: about 220 m for provinces down to 20 m for GN
// divisions. The collector applies them to shared edges.
var Tolerances = map[Level]float64{
	LevelProvince: 0.002,
	LevelDistrict: 0.001,
	LevelDS:       0.0005,
	LevelGN:       0.0002,
}

// Source identifiers, as listed in docs/data-sources.md.
const (
	// BoundarySource is nuuuwan/lk_admin_regions.
	BoundarySource = "nuuuwan-lk-admin-regions"
	// PopulationSource is nuuuwan/lk_census_2024.
	PopulationSource = "nuuuwan-lk-census-2024"
)

// Attribution credits the boundary and population data.
const Attribution = "Survey Department of Sri Lanka and Department of Census and Statistics, " +
	"via OCHA HDX (CC BY-IGO) and nuuuwan/lk_admin_regions; population: Census of Population and Housing 2024"
