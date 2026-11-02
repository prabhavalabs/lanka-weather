package osmreservoirs

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Target is a reservoir to find on the map.
type Target struct {
	ID, Name string
	// District is the source's district, if it gives one.
	District *string
	// Kind is major, medium or mahaweli.
	Kind string
}

// Measured is a candidate with what the database worked out about its shape.
type Measured struct {
	OSMID string
	// Name is the candidate's display name, Kind water or dam, Names its tags.
	Name  string
	Kind  string
	Names []string
	// Lon and Lat are a point on the polygon, or on the dam's line.
	Lon, Lat float64
	// AreaHa is the polygon's area in hectares, 0 when it has none.
	AreaHa float64
	// Districts are the districts the shape touches, as named in our regions;
	// NearDistricts are those within about 15 km of it.
	Districts     []string
	NearDistricts []string
}

// Result is a reservoir with the candidate chosen for it, or the reason none
// was. Result.Candidate is nil when the reservoir is not mapped.
type Result struct {
	Target    Target
	Candidate *Measured
	// Reason says why a reservoir was not mapped: no-name-match, no-district,
	// wrong-district, too-small or ambiguous.
	Reason string
	// Near says the match lies just outside the listed district: see Match.
	Near bool
}

// Reasons a reservoir is left off the map.
const (
	ReasonNoNameMatch   = "no-name-match"
	ReasonNoDistrict    = "no-district"
	ReasonWrongDistrict = "wrong-district"
	ReasonTooSmall      = "too-small"
	ReasonAmbiguous     = "ambiguous"
)

// genericWords describe what a water body is, not which one, and are dropped
// when names are compared. Common misspellings in OpenStreetMap are included.
var genericWords = map[string]bool{
	"wewa": true, "wawa": true, "weva": true, "vewa": true, "tank": true, "tanks": true,
	"reservoir": true, "reservior": true, "rerservoir": true, "resevoir": true, "resrvoir": true,
	"lake": true, "kulam": true, "dam": true, "dams": true, "anicut": true,
	"samudraya": true, "samudra": true, "sagaraya": true, "sagara": true,
}

// genericSuffixes are the same words written as one with the name.
var genericSuffixes = []string{"wewa", "wawa", "weva", "vewa", "kulam"}

// Key reduces a name to what identifies a water body: lower case, letters only,
// without generic words, and with the romanisation differences common in
// Sinhala and Tamil names folded (th/t, dh/d, v/w, doubled letters). Names that
// are only generic words have no key.
func Key(name string) string {
	var cleaned strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r == '\'' || r == '’' || r == '`':
		case unicode.IsLetter(r) && r < unicode.MaxASCII:
			cleaned.WriteRune(r)
		default:
			cleaned.WriteRune(' ')
		}
	}
	var kept []string
	for _, word := range strings.Fields(cleaned.String()) {
		if genericWords[word] {
			continue
		}
		for _, suffix := range genericSuffixes {
			if strings.HasSuffix(word, suffix) && len(word) >= len(suffix)+3 {
				word = strings.TrimSuffix(word, suffix)
				break
			}
		}
		kept = append(kept, word)
	}
	compact := strings.Join(kept, "")
	if compact == "" {
		return ""
	}
	compact = strings.NewReplacer("th", "t", "dh", "d", "ph", "p", "v", "w").Replace(compact)
	var folded strings.Builder
	var previous rune
	for _, r := range compact {
		if r != previous {
			folded.WriteRune(r)
		}
		previous = r
	}
	return folded.String()
}

// Keys returns every key a target can match, from its name and its aliases.
func Keys(target Target) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range append([]string{target.Name}, Aliases[target.ID]...) {
		if key := Key(name); key != "" && !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}

// CandidateKeys returns every key a candidate's names reduce to.
func CandidateKeys(names []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range names {
		if key := Key(name); key != "" && !seen[key] {
			seen[key] = true
			out = append(out, key)
		}
	}
	return out
}

// minimumHectares is the smallest polygon accepted for a kind of reservoir:
// the sheets list irrigation schemes, so a garden pond with the same name is
// not one of them. Some Mahaweli reservoirs are small balancing ponds, but each
// has a hand-checked district.
func minimumHectares(kind string) float64 {
	if kind == "major" {
		return 10
	}
	return 2
}

// splitBodyKm is how close a smaller polygon with the same name must lie to the
// largest for the two to be parts of one reservoir rather than two tanks.
const splitBodyKm = 4.0

// dominance is how many times larger the largest polygon must be than a
// same-named one elsewhere in the district to be taken as the reservoir.
const dominance = 4.0

// Match chooses at most one candidate per target. A candidate must carry a name
// that reduces to the target's key, and lie in the target's district: the
// source's, or those in Districts for a reservoir the source gives none for.
// A reservoir with no known district is not mapped, however well its name fits.
// Among water bodies the largest wins if it is much larger than the others or
// they are parts of one body; a dam marks a reservoir only if no water body
// qualifies and it is the only one with that name there.
//
// The feed's district is sometimes the Irrigation Department's range office
// rather than the district on the ground: Bomuruella is listed under Badulla and
// lies just inside Nuwara Eliya. So a candidate within about 15 km of the district is
// also accepted, but only if nothing else in the whole island qualifies under
// that name. A same-named tank in a far district is therefore never taken, and
// two same-named tanks are never told apart by distance. Match does no I/O.
func Match(targets []Target, candidates []Measured) []Result {
	index := map[string][]int{}
	for position, candidate := range candidates {
		for _, key := range CandidateKeys(candidate.Names) {
			index[key] = append(index[key], position)
		}
	}
	results := make([]Result, 0, len(targets))
	for _, target := range targets {
		results = append(results, matchOne(target, candidates, index))
	}
	return results
}

func matchOne(target Target, candidates []Measured, index map[string][]int) Result {
	seen := map[int]bool{}
	var named []Measured
	for _, key := range Keys(target) {
		for _, position := range index[key] {
			if !seen[position] {
				seen[position] = true
				named = append(named, candidates[position])
			}
		}
	}
	result := Result{Target: target}
	if len(named) == 0 {
		result.Reason = ReasonNoNameMatch
		return result
	}
	allowed := districtsOf(target)
	if len(allowed) == 0 {
		result.Reason = ReasonNoDistrict
		return result
	}
	var waters, dams, qualifying []Measured
	smallOnly := false
	for _, candidate := range named {
		large := candidate.Kind == KindWater && candidate.AreaHa >= minimumHectares(target.Kind)
		if large || candidate.Kind == KindDam {
			qualifying = append(qualifying, candidate)
		}
		if !inDistricts(candidate.Districts, allowed) {
			continue
		}
		switch {
		case candidate.Kind == KindDam:
			dams = append(dams, candidate)
		case large:
			waters = append(waters, candidate)
		default:
			smallOnly = true
		}
	}
	if len(waters) > 0 {
		sort.SliceStable(waters, func(i, j int) bool { return waters[i].AreaHa > waters[j].AreaHa })
		largest := waters[0]
		for _, other := range waters[1:] {
			if largest.AreaHa < dominance*other.AreaHa && kmBetween(largest, other) > splitBodyKm {
				// Two tanks of similar size and the same name: fall back to a
				// dam only if the reservoir has an unambiguous one.
				if len(dams) == 1 {
					result.Candidate = &dams[0]
					return result
				}
				result.Reason = ReasonAmbiguous
				return result
			}
		}
		result.Candidate = &largest
		return result
	}
	switch {
	case len(dams) == 1:
		result.Candidate = &dams[0]
	case len(dams) > 1:
		result.Reason = ReasonAmbiguous
	case smallOnly:
		result.Reason = ReasonTooSmall
	case len(qualifying) == 1 && inDistricts(qualifying[0].NearDistricts, allowed):
		result.Candidate, result.Near = &qualifying[0], true
	default:
		result.Reason = ReasonWrongDistrict
	}
	return result
}

func districtsOf(target Target) []string {
	if target.District != nil && *target.District != "" {
		return []string{canonicalDistrict(*target.District)}
	}
	return Districts[target.ID]
}

func inDistricts(districts, allowed []string) bool {
	for _, district := range districts {
		for _, want := range allowed {
			if canonicalDistrict(district) == canonicalDistrict(want) {
				return true
			}
		}
	}
	return false
}

// canonicalDistrict reconciles the source's spelling of a district with ours.
func canonicalDistrict(name string) string {
	if strings.EqualFold(strings.TrimSpace(name), "Moneragala") {
		return "Monaragala"
	}
	return strings.TrimSpace(name)
}

func kmBetween(a, b Measured) float64 {
	const earth = 6371.0
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(a.Lat*math.Pi/180)*math.Cos(b.Lat*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earth * math.Asin(math.Sqrt(h))
}
