package rivernet

import (
	"sort"
	"strconv"
	"strings"
)

// Candidate is a river segment near a gauge.
type Candidate struct {
	SegmentID, MainRiv, HybasL12 int64
	UplandKM2, DistanceM         float64
}

const (
	// stationLimitM is how far a station may lie from a line. HydroRIVERS
	// follows a 15 arc-second grid, about 460 m, and a gauge can stand on a
	// bank or a bridge; anything beyond 3 km is on some other water.
	stationLimitM = 3000
	// googleLimitM is wider: a Google gauge is a virtual point in a sub-basin,
	// and the sub-basin id does most of the work.
	googleLimitM = 5000
	// minGaugeUplandKM2 keeps a gauge on a river rather than a rill beside it.
	minGaugeUplandKM2 = 15
	// tieMarginM is the difference in distance too small to mean anything at
	// this resolution, within which the larger river is taken: gauges stand on
	// rivers, and a small stream happens to run closer to some of them.
	tieMarginM = 500
)

// StationHint is what is known about a station's place besides its position.
type StationHint struct {
	// System is the river system its basin name means, or 0.
	System int64
	// MainStem holds the main stem's segments when the department's list says
	// the station stands on the basin's main river. HydroRIVERS' line can run
	// a kilometre or two from the true channel near the coast: Nagalagam Street
	// lies 37 m from a canal and 2 km from this line of the Kelani.
	MainStem map[int64]bool
}

// ChooseStationSegment picks the segment a station stands on: the nearest
// suitable one, preferring the main stem if it is known to stand on it, then
// its own river system. A station whose basin is unknown or matches nothing
// nearby is placed by position alone (method "position"). It reports false when
// nothing suitable lies within the limit.
func ChooseStationSegment(candidates []Candidate, hint StationHint) (Candidate, string, bool) {
	expectedSystem := hint.System
	usable := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.DistanceM <= stationLimitM && candidate.UplandKM2 >= minGaugeUplandKM2 {
			usable = append(usable, candidate)
		}
	}
	if hint.MainStem != nil {
		var stem []Candidate
		for _, candidate := range usable {
			if hint.MainStem[candidate.SegmentID] {
				stem = append(stem, candidate)
			}
		}
		if chosen, ok := largestWithinTie(stem); ok {
			return chosen, "main-stem", true
		}
	}
	if expectedSystem != 0 {
		var own []Candidate
		for _, candidate := range usable {
			if candidate.MainRiv == expectedSystem {
				own = append(own, candidate)
			}
		}
		if len(own) > 0 {
			chosen, _ := largestWithinTie(own)
			return chosen, "basin", true
		}
	}
	if chosen, ok := largestWithinTie(usable); ok {
		return chosen, "position", true
	}
	return Candidate{}, "", false
}

// ChooseGoogleSegment picks the segment for a Google gauge. Its id names a
// HydroBASINS sub-basin, and the forecast is for the river leaving it, so the
// largest segment of that sub-basin nearby wins. Without one, the nearest
// suitable segment will do.
func ChooseGoogleSegment(candidates []Candidate, hybas int64) (Candidate, string, bool) {
	var inBasin []Candidate
	for _, candidate := range candidates {
		if hybas != 0 && candidate.HybasL12 == hybas && candidate.DistanceM <= googleLimitM {
			inBasin = append(inBasin, candidate)
		}
	}
	if len(inBasin) > 0 {
		sort.SliceStable(inBasin, func(i, j int) bool { return inBasin[i].UplandKM2 > inBasin[j].UplandKM2 })
		return inBasin[0], "hybas", true
	}
	var usable []Candidate
	for _, candidate := range candidates {
		if candidate.DistanceM <= googleLimitM && candidate.UplandKM2 >= minGaugeUplandKM2 {
			usable = append(usable, candidate)
		}
	}
	if chosen, ok := largestWithinTie(usable); ok {
		return chosen, "position", true
	}
	return Candidate{}, "", false
}

func largestWithinTie(candidates []Candidate) (Candidate, bool) {
	if len(candidates) == 0 {
		return Candidate{}, false
	}
	nearest := candidates[0].DistanceM
	for _, candidate := range candidates {
		if candidate.DistanceM < nearest {
			nearest = candidate.DistanceM
		}
	}
	var best Candidate
	found := false
	for _, candidate := range candidates {
		if candidate.DistanceM > nearest+tieMarginM {
			continue
		}
		if !found || candidate.UplandKM2 > best.UplandKM2 ||
			(candidate.UplandKM2 == best.UplandKM2 && candidate.SegmentID < best.SegmentID) {
			best, found = candidate, true
		}
	}
	return best, found
}

// HybasOfGauge reads the sub-basin from a Google gauge id, "hybas_4121599890".
func HybasOfGauge(id string) int64 {
	digits, ok := strings.CutPrefix(id, "hybas_")
	if !ok {
		return 0
	}
	value, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// StreamVote says a segment runs along a named stream of the department's map.
type StreamVote struct {
	SegmentID int64
	Name      string
}

const (
	// minNamedChainKM keeps only names that follow a river for a real distance;
	// the map's small polygons beside a line are not the line's name.
	minNamedChainKM = 8
	// minNamedUplandKM2 is the size a tributary must reach to be named this way.
	minNamedUplandKM2 = 100
)

// TributariesFromVotes turns the named stream polygons into named tributaries.
// For each name in each system it takes the longest connected chain of segments
// voted for that name. A name that many separate streams share, such as Kuda
// Oya ("small river"), still names one chain at most per system, and only if
// that chain is long enough to trust. Each result runs from the top of its
// chain, carried on up the largest branch, to the last segment voted for it.
func TributariesFromVotes(segments []Segment, votes []StreamVote, mainNames map[int64]string) []Tributary {
	byID := make(map[int64]*Segment, len(segments))
	upstream := map[int64][]int64{}
	for index := range segments {
		segment := &segments[index]
		byID[segment.ID] = segment
		if segment.NextDown != 0 {
			upstream[segment.NextDown] = append(upstream[segment.NextDown], segment.ID)
		}
	}
	for _, list := range upstream {
		sort.Slice(list, func(i, j int) bool {
			if byID[list[i]].UplandKM2 != byID[list[j]].UplandKM2 {
				return byID[list[i]].UplandKM2 > byID[list[j]].UplandKM2
			}
			return list[i] < list[j]
		})
	}
	type key struct {
		system int64
		name   string
	}
	groups := map[key]map[int64]bool{}
	for _, vote := range votes {
		segment, ok := byID[vote.SegmentID]
		if !ok {
			continue
		}
		group := key{system: segment.MainRiv, name: vote.Name}
		if groups[group] == nil {
			groups[group] = map[int64]bool{}
		}
		groups[group][vote.SegmentID] = true
	}
	var tributaries []Tributary
	for group, members := range groups {
		if Fold(mainNames[group.system]) == Fold(group.name) {
			continue
		}
		var bestTop, bestEnd int64
		bestLength, bestUpland := 0.0, 0.0
		for id := range members {
			// Start only at the lower end of a chain.
			if members[byID[id].NextDown] {
				continue
			}
			length, top := byID[id].LengthKM, id
			for {
				next := int64(0)
				for _, candidate := range upstream[top] {
					if members[candidate] {
						next = candidate
						break
					}
				}
				if next == 0 {
					break
				}
				length += byID[next].LengthKM
				top = next
			}
			if length > bestLength || (length == bestLength && id < bestTop) {
				bestLength, bestTop, bestEnd, bestUpland = length, top, id, byID[id].UplandKM2
			}
		}
		if bestLength < minNamedChainKM || bestUpland < minNamedUplandKM2 {
			continue
		}
		for {
			list := upstream[bestTop]
			if len(list) == 0 || byID[list[0]].UplandKM2 < minStemUplandKM2 {
				break
			}
			bestTop = list[0]
		}
		tributaries = append(tributaries, Tributary{SourceID: bestTop, EndID: bestEnd, Name: group.name})
	}
	sort.Slice(tributaries, func(i, j int) bool { return tributaries[i].SourceID < tributaries[j].SourceID })
	return tributaries
}

// SourceAbove walks from a segment up the largest branch to where the stream
// gets too small to be a stem: the source to give a tributary whose gauge
// stands on that segment.
func SourceAbove(segments []Segment, from int64) int64 {
	byID := make(map[int64]*Segment, len(segments))
	upstream := map[int64][]int64{}
	for index := range segments {
		segment := &segments[index]
		byID[segment.ID] = segment
		if segment.NextDown != 0 {
			upstream[segment.NextDown] = append(upstream[segment.NextDown], segment.ID)
		}
	}
	for current := from; ; {
		var best int64
		for _, candidate := range upstream[current] {
			if byID[candidate].UplandKM2 >= minStemUplandKM2 &&
				(best == 0 || byID[candidate].UplandKM2 > byID[best].UplandKM2) {
				best = candidate
			}
		}
		if best == 0 {
			return current
		}
		current = best
	}
}

// MainStems returns, for each system's outlet, the segments of its main stem:
// the chain up the largest branch.
func MainStems(segments []Segment) map[int64]map[int64]bool {
	byID := make(map[int64]*Segment, len(segments))
	upstream := map[int64][]int64{}
	for index := range segments {
		segment := &segments[index]
		byID[segment.ID] = segment
		if segment.NextDown != 0 {
			upstream[segment.NextDown] = append(upstream[segment.NextDown], segment.ID)
		}
	}
	stems := map[int64]map[int64]bool{}
	for index := range segments {
		segment := &segments[index]
		if segment.NextDown != 0 {
			continue
		}
		stem := map[int64]bool{}
		for current := segment.ID; current != 0; {
			stem[current] = true
			var best int64
			for _, candidate := range upstream[current] {
				if byID[candidate].UplandKM2 >= minStemUplandKM2 &&
					(best == 0 || byID[candidate].UplandKM2 > byID[best].UplandKM2) {
					best = candidate
				}
			}
			current = best
		}
		stems[segment.MainRiv] = stem
	}
	return stems
}
