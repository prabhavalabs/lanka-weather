package riverrisk

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivernet"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// reservoirSearchDegrees is how far around a reservoir segments are read,
	// about 3.3 km; reservoirLimitM decides from among them. A reservoir further
	// than three cells of the 15 arc-second grid from any line is not on it.
	reservoirSearchDegrees = 0.03
	reservoirLimitM        = 1500
	// minOutletUplandKM2 keeps a reservoir on a river rather than a rill beside
	// it, as for gauges (ADR 0010).
	minOutletUplandKM2 = 15
	// directlyDownstreamKM is how far below a reservoir a stretch counts as
	// directly downstream: a flood wave of 1 to 2 m/s covers it in about the six
	// hours that are the estimate's lead time.
	directlyDownstreamKM = 30
)

// Ways a reservoir comes to be linked.
const (
	LinkByPosition = "position"
	LinkByRiver    = "river"
)

// Why a reservoir is not linked.
const (
	UnlinkedNoPlace   = "no_place"
	UnlinkedTooFar    = "too_far"
	UnlinkedConflict  = "conflict"
	UnlinkedNoStretch = "no_stretch"
	UnlinkedAmbiguous = "ambiguous"
)

// ReservoirCandidate is a river segment near a mapped reservoir.
type ReservoirCandidate struct {
	SegmentID int64
	UplandKM2 float64
	// Inside says the reservoir's polygon touches the segment.
	Inside    bool
	DistanceM float64
}

// ReservoirPlace is what is known about where a reservoir is.
type ReservoirPlace struct {
	ID, Name string
	// River and District are the source's words, if any. River is often the
	// basin or the managing agency's system rather than the stream it spills to.
	River, District *string
	Mapped          bool
	Candidates      []ReservoirCandidate
}

// LinkSegment is a river segment as the link needs it.
type LinkSegment struct {
	ID, NextDown, MainRiv int64
	// ReachFID is the stretch it belongs to, or 0 for a stream not drawn.
	ReachFID                          int64
	LengthKM, UplandKM2, DischargeCMS float64
}

// LinkReach is a stretch as the link needs it.
type LinkReach struct {
	FID       int64
	RiverID   string
	Gauged    bool
	Districts []string
}

// LinkRiver names a river and the system it belongs to.
type LinkRiver struct {
	ID      string
	Names   []string
	MainRiv int64
}

// LinkedReach is a stretch that a reservoir's water reaches directly.
type LinkedReach struct {
	FID int64
	// Into says the reservoir empties into this stretch.
	Into bool
	// RiverCMS and RiverUplandKM2 describe the river where the water enters.
	RiverCMS, RiverUplandKM2 float64
}

// ReservoirLink is one reservoir's place on the network.
type ReservoirLink struct {
	ID string
	// Method is position, or river when only the source's river placed it.
	Method  string
	Reaches []LinkedReach
}

// UnlinkedReservoir says why a reservoir has no place on the network. It then
// counts for nothing: never guessed onto a river.
type UnlinkedReservoir struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

// LinkReport says how many reservoirs could be placed on the network.
type LinkReport struct {
	ByPosition int                 `json:"byPosition"`
	ByRiver    int                 `json:"byRiver"`
	Unlinked   []UnlinkedReservoir `json:"unlinked"`
}

// LinkReservoirs places reservoirs on the network, and says why for those it
// cannot. A position is snapped to the largest segment its polygon touches, or
// failing that to one close by, and followed down NEXT_DOWN. A reservoir with no
// position is linked by the source's river only if that leaves one stretch: that
// river's only one, or the only one in its district. Pure.
func LinkReservoirs(places []ReservoirPlace, segments []LinkSegment, reaches []LinkReach, rivers []LinkRiver) ([]ReservoirLink, []UnlinkedReservoir) {
	segmentByID := make(map[int64]*LinkSegment, len(segments))
	for index := range segments {
		segmentByID[segments[index].ID] = &segments[index]
	}
	// A segment with another of its own stretch above it is not the stretch's head.
	joined := make(map[int64]bool, len(segments))
	for _, segment := range segments {
		if next, ok := segmentByID[segment.NextDown]; ok && segment.ReachFID != 0 && next.ReachFID == segment.ReachFID {
			joined[next.ID] = true
		}
	}
	reachByFID := make(map[int64]*LinkReach, len(reaches))
	reachesOfRiver := map[string][]*LinkReach{}
	for index := range reaches {
		reach := &reaches[index]
		reachByFID[reach.FID] = reach
		reachesOfRiver[reach.RiverID] = append(reachesOfRiver[reach.RiverID], reach)
	}
	byName := map[string][]LinkRiver{}
	for _, river := range rivers {
		for _, name := range river.Names {
			key := rivernet.CanonicalBasin(name)
			byName[key] = append(byName[key], river)
		}
	}
	// A stretch's flow where it ends, for a reservoir placed only by its river:
	// where along the stretch it lies is not known, so the largest figure is used.
	lowest := map[int64]*LinkSegment{}
	for index := range segments {
		segment := &segments[index]
		if segment.ReachFID != 0 && (lowest[segment.ReachFID] == nil || segment.UplandKM2 > lowest[segment.ReachFID].UplandKM2) {
			lowest[segment.ReachFID] = segment
		}
	}

	var links []ReservoirLink
	var unlinked []UnlinkedReservoir
	refuse := func(place ReservoirPlace, reason, detail string) {
		unlinked = append(unlinked, UnlinkedReservoir{ID: place.ID, Name: place.Name, Reason: reason, Detail: detail})
	}
	for _, place := range places {
		named := listedRivers(place.River, byName)
		if !place.Mapped {
			if len(named) == 0 {
				refuse(place, UnlinkedNoPlace, "No position, and no river the network knows.")
				continue
			}
			options := []*LinkReach{}
			for _, river := range named {
				options = append(options, reachesOfRiver[river.ID]...)
			}
			if len(options) > 1 {
				options = inDistrict(options, place.District)
			}
			if len(options) != 1 {
				refuse(place, UnlinkedAmbiguous, fmt.Sprintf("No position, and %s does not say which stretch.", text(place.River)))
				continue
			}
			reach := LinkedReach{FID: options[0].FID}
			if end := lowest[options[0].FID]; end != nil {
				reach.RiverCMS, reach.RiverUplandKM2 = end.DischargeCMS, end.UplandKM2
			}
			links = append(links, ReservoirLink{ID: place.ID, Method: LinkByRiver, Reaches: []LinkedReach{reach}})
			continue
		}
		outlet, ok := chooseOutlet(place.Candidates)
		if !ok {
			refuse(place, UnlinkedTooFar, "No river close enough to its position.")
			continue
		}
		segment := segmentByID[outlet.SegmentID]
		if segment == nil {
			refuse(place, UnlinkedTooFar, "Its nearest river is no longer in the network.")
			continue
		}
		if len(named) > 0 && !sameSystem(named, segment.MainRiv) {
			refuse(place, UnlinkedConflict, fmt.Sprintf("Its position is not on %s, the river the source names.", text(place.River)))
			continue
		}
		reached := walkDown(segment, segmentByID, reachByFID, joined)
		if len(reached) == 0 {
			refuse(place, UnlinkedNoStretch, "It drains to a stream with no stretch drawn near enough.")
			continue
		}
		links = append(links, ReservoirLink{ID: place.ID, Method: LinkByPosition, Reaches: reached})
	}
	return links, unlinked
}

func chooseOutlet(candidates []ReservoirCandidate) (ReservoirCandidate, bool) {
	var inside *ReservoirCandidate
	var near []rivernet.Candidate
	for index := range candidates {
		candidate := &candidates[index]
		if candidate.UplandKM2 < minOutletUplandKM2 {
			continue
		}
		if candidate.Inside && (inside == nil || candidate.UplandKM2 > inside.UplandKM2) {
			inside = candidate
		}
		if candidate.DistanceM <= reservoirLimitM {
			near = append(near, rivernet.Candidate{SegmentID: candidate.SegmentID, UplandKM2: candidate.UplandKM2, DistanceM: candidate.DistanceM})
		}
	}
	if inside != nil {
		return *inside, true
	}
	chosen, _, ok := rivernet.ChooseStationSegment(near, rivernet.StationHint{})
	if !ok {
		return ReservoirCandidate{}, false
	}
	return ReservoirCandidate{SegmentID: chosen.SegmentID, UplandKM2: chosen.UplandKM2, DistanceM: chosen.DistanceM}, true
}

// walkDown follows the water from a reservoir. It stops at the distance limit,
// the sea, or where it reaches a gauge: a stretch that starts at a gauge is
// measured below the spill, so that gauge speaks for it. A stream joining part
// way down a gauged stretch is below that gauge, which cannot see it.
func walkDown(start *LinkSegment, segments map[int64]*LinkSegment, reaches map[int64]*LinkReach, joined map[int64]bool) []LinkedReach {
	var out []LinkedReach
	walked := 0.0
	seen := map[int64]bool{}
	for current := start; current != nil && !seen[current.ID] && walked <= directlyDownstreamKM; current = segments[current.NextDown] {
		seen[current.ID] = true
		walked += current.LengthKM
		if current.ReachFID == 0 || (len(out) > 0 && out[len(out)-1].FID == current.ReachFID) {
			continue
		}
		reach := reaches[current.ReachFID]
		if reach == nil {
			continue
		}
		if current != start && reach.Gauged && !joined[current.ID] {
			break
		}
		out = append(out, LinkedReach{FID: reach.FID, Into: current == start, RiverCMS: current.DischargeCMS, RiverUplandKM2: current.UplandKM2})
	}
	return out
}

// listedRivers resolves the source's river field, which can name two ("Andal Oya
// & Navakiri Ara") and can name a river the network does not know.
func listedRivers(field *string, byName map[string][]LinkRiver) []LinkRiver {
	if field == nil {
		return nil
	}
	var out []LinkRiver
	for _, part := range strings.FieldsFunc(strings.ReplaceAll(*field, " and ", "&"), func(r rune) bool { return r == '&' || r == ',' }) {
		out = append(out, byName[rivernet.CanonicalBasin(part)]...)
	}
	return out
}

func sameSystem(rivers []LinkRiver, system int64) bool {
	for _, river := range rivers {
		if river.MainRiv == system {
			return true
		}
	}
	return false
}

func inDistrict(options []*LinkReach, district *string) []*LinkReach {
	if district == nil {
		return nil
	}
	var out []*LinkReach
	for _, reach := range options {
		for _, name := range reach.Districts {
			if strings.EqualFold(name, *district) {
				out = append(out, reach)
				break
			}
		}
	}
	return out
}

func text(value *string) string {
	if value == nil {
		return "the source"
	}
	return *value
}

// linkState is what the last link found, reused until its inputs change.
type linkState struct {
	key    string
	links  []ReservoirLink
	report LinkReport
}

// loadLinks links reservoirs to the network, once per version of the network and
// of the reservoirs' map. Reservoir links are logged when they change.
func (service *Service) loadLinks(ctx context.Context, queries *store.Queries, net *network, layer reservoirs.FeatureCollection) (*linkState, error) {
	key := fmt.Sprintf("%s|%d|%d", net.version, layer.Meta.Counts.Mapped, len(layer.Features))
	if layer.Meta.MapTime != nil {
		key += "|" + layer.Meta.MapTime.Format(time.RFC3339)
	}
	if service.links != nil && service.links.key == key {
		return service.links, nil
	}
	rows, err := queries.ReservoirSegmentCandidates(ctx, reservoirSearchDegrees)
	if err != nil {
		return nil, fmt.Errorf("read reservoir segments: %w", err)
	}
	candidates := map[string][]ReservoirCandidate{}
	for _, row := range rows {
		candidates[row.ReservoirID] = append(candidates[row.ReservoirID], ReservoirCandidate{SegmentID: row.HyrivID, UplandKM2: row.UplandKm2, Inside: row.Inside, DistanceM: row.DistanceM})
	}
	districts, err := queries.ReachDistrictNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("read stretches' districts: %w", err)
	}
	reachDistricts := map[int64][]string{}
	for _, row := range districts {
		reachDistricts[row.ReachFid] = append(reachDistricts[row.ReachFid], row.Name)
	}
	places := make([]ReservoirPlace, 0, len(layer.Features))
	for _, feature := range layer.Features {
		properties := feature.Properties
		places = append(places, ReservoirPlace{ID: properties.ID, Name: properties.Name, River: properties.River,
			District: properties.District, Mapped: properties.Mapped, Candidates: candidates[properties.ID]})
	}
	segments := make([]LinkSegment, len(net.topology))
	for index, row := range net.topology {
		segment := LinkSegment{ID: row.HyrivID, NextDown: row.NextDown, MainRiv: row.MainRiv, LengthKM: row.LengthKm,
			UplandKM2: row.UplandKm2, DischargeCMS: row.DischargeCms}
		if row.ReachFid != nil {
			segment.ReachFID = *row.ReachFid
		}
		segments[index] = segment
	}
	reaches := make([]LinkReach, len(net.reaches))
	for index, row := range net.reaches {
		reaches[index] = LinkReach{FID: row.Fid, RiverID: row.RiverID, Gauged: row.Gauged, Districts: reachDistricts[row.Fid]}
	}
	rivers := make([]LinkRiver, 0, len(net.rivers))
	for _, row := range net.rivers {
		rivers = append(rivers, LinkRiver{ID: row.ID, Names: append([]string{row.Name}, row.Aliases...), MainRiv: row.MainRiv})
	}
	links, unlinked := LinkReservoirs(places, segments, reaches, rivers)
	state := &linkState{key: key, links: links, report: LinkReport{Unlinked: unlinked}}
	if state.report.Unlinked == nil {
		state.report.Unlinked = []UnlinkedReservoir{}
	}
	sort.Slice(state.report.Unlinked, func(i, j int) bool { return state.report.Unlinked[i].ID < state.report.Unlinked[j].ID })
	names := make([]string, 0, len(unlinked))
	for _, link := range links {
		if link.Method == LinkByRiver {
			state.report.ByRiver++
		} else {
			state.report.ByPosition++
		}
	}
	for _, entry := range state.report.Unlinked {
		names = append(names, entry.ID+": "+entry.Reason)
	}
	if service.Logger != nil {
		service.Logger.Info("reservoirs linked to the river network", "position", state.report.ByPosition,
			"river", state.report.ByRiver, "unlinked", len(unlinked), "unlinked_reservoirs", names)
	}
	service.links = state
	return state, nil
}

// reachReservoirs turns the links and the latest readings into each stretch's
// inputs.
func reachReservoirs(links []ReservoirLink, layer reservoirs.FeatureCollection) map[int64][]Reservoir {
	byID := make(map[string]reservoirs.Properties, len(layer.Features))
	for _, feature := range layer.Features {
		byID[feature.Properties.ID] = feature.Properties
	}
	out := map[int64][]Reservoir{}
	for _, link := range links {
		properties, ok := byID[link.ID]
		if !ok {
			continue
		}
		base := Reservoir{Name: properties.Name, Status: properties.Status, Kind: properties.Type, Stale: properties.Stale, Listed: link.Method == LinkByRiver}
		if flow := properties.Flow; flow != nil {
			base.FlowKind, base.Cusecs, base.FlowStale = flow.Kind, flow.Cusecs, flow.Stale
		}
		for _, reach := range link.Reaches {
			entry := base
			entry.Into, entry.RiverCMS, entry.RiverUplandKM2 = reach.Into, reach.RiverCMS, reach.RiverUplandKM2
			out[reach.FID] = append(out[reach.FID], entry)
		}
	}
	return out
}
