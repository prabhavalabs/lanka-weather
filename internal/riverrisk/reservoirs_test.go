package riverrisk

import (
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/floods"
	"github.com/prabhavalabs/lanka-weather/internal/reservoirs"
	"github.com/prabhavalabs/lanka-weather/internal/rivers"
)

// Two systems. The stream 1→2→3 joins the main stem 10→11→12→13 at 12. Stretch
// 100 is the stream, 200 the main stem above a gauge, 300 below it, 400 at the
// mouth. Segment 5 is a stream with no stretch that runs into segment 2.
func linkFixture() ([]LinkSegment, []LinkReach, []LinkRiver) {
	segments := []LinkSegment{
		{ID: 1, NextDown: 2, MainRiv: 13, ReachFID: 100, LengthKM: 5, UplandKM2: 50, DischargeCMS: 1},
		{ID: 2, NextDown: 3, MainRiv: 13, ReachFID: 100, LengthKM: 5, UplandKM2: 80, DischargeCMS: 2},
		{ID: 3, NextDown: 12, MainRiv: 13, ReachFID: 100, LengthKM: 5, UplandKM2: 100, DischargeCMS: 3},
		{ID: 5, NextDown: 2, MainRiv: 13, LengthKM: 4, UplandKM2: 20, DischargeCMS: 0.5},
		{ID: 10, NextDown: 11, MainRiv: 13, ReachFID: 200, LengthKM: 10, UplandKM2: 400, DischargeCMS: 20},
		{ID: 11, NextDown: 12, MainRiv: 13, ReachFID: 300, LengthKM: 10, UplandKM2: 500, DischargeCMS: 25},
		{ID: 12, NextDown: 13, MainRiv: 13, ReachFID: 300, LengthKM: 10, UplandKM2: 700, DischargeCMS: 35},
		{ID: 13, NextDown: 14, MainRiv: 13, ReachFID: 400, LengthKM: 40, UplandKM2: 900, DischargeCMS: 45},
		{ID: 14, NextDown: 0, MainRiv: 13, ReachFID: 500, LengthKM: 10, UplandKM2: 950, DischargeCMS: 46},
		{ID: 90, NextDown: 0, MainRiv: 90, ReachFID: 900, LengthKM: 10, UplandKM2: 120, DischargeCMS: 3},
	}
	reaches := []LinkReach{
		{FID: 100, RiverID: "stream", Districts: []string{"Matale"}},
		{FID: 200, RiverID: "main", Districts: []string{"Kandy"}},
		{FID: 300, RiverID: "main", Gauged: true, Districts: []string{"Kandy", "Matale"}},
		{FID: 400, RiverID: "main", Gauged: true, Districts: []string{"Ampara"}},
		{FID: 500, RiverID: "main", Gauged: true, Districts: []string{"Ampara"}},
		{FID: 900, RiverID: "other", Districts: []string{"Galle"}},
	}
	rivers := []LinkRiver{
		{ID: "stream", Names: []string{"Stream Oya"}, MainRiv: 13},
		{ID: "main", Names: []string{"Main Ganga", "Old Name"}, MainRiv: 13},
		{ID: "other", Names: []string{"Other Oya"}, MainRiv: 90},
	}
	return segments, reaches, rivers
}

func str(value string) *string { return &value }

func inside(segment int64, upland float64) ReservoirCandidate {
	return ReservoirCandidate{SegmentID: segment, UplandKM2: upland, Inside: true}
}

func link(t *testing.T, place ReservoirPlace) (ReservoirLink, UnlinkedReservoir) {
	t.Helper()
	segments, reaches, rivers := linkFixture()
	links, unlinked := LinkReservoirs([]ReservoirPlace{place}, segments, reaches, rivers)
	switch {
	case len(links) == 1:
		return links[0], UnlinkedReservoir{}
	case len(unlinked) == 1:
		return ReservoirLink{}, unlinked[0]
	}
	t.Fatalf("links %v unlinked %v", links, unlinked)
	return ReservoirLink{}, UnlinkedReservoir{}
}

func fids(link ReservoirLink) []int64 {
	out := []int64{}
	for _, reach := range link.Reaches {
		out = append(out, reach.FID)
	}
	return out
}

func sameFIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func TestLinkByPositionWalksDownToTheNextGauge(t *testing.T) {
	cases := []struct {
		name       string
		candidates []ReservoirCandidate
		want       []int64
	}{
		// A stream joining part way down a gauged stretch is below that gauge.
		{"on a stream into a gauged stretch", []ReservoirCandidate{inside(1, 50)}, []int64{100, 300}},
		// The stretch below the gauge starts at the gauge, which sees the water.
		{"above a gauge stops at the gauge", []ReservoirCandidate{inside(10, 400)}, []int64{200}},
		{"the polygon's largest segment is its outlet", []ReservoirCandidate{inside(10, 400), inside(1, 50)}, []int64{200}},
		{"from a stream with no stretch", []ReservoirCandidate{inside(5, 20)}, []int64{100, 300}},
		// 30 km is walked: 11 and 12 are 20 km, and 13 is 40 km more but starts inside it.
		{"a walk ends at its limit", []ReservoirCandidate{inside(11, 500)}, []int64{300}},
		{"a dam close to a river", []ReservoirCandidate{{SegmentID: 1, UplandKM2: 50, DistanceM: 1400}}, []int64{100, 300}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, refused := link(t, ReservoirPlace{ID: "r", Name: "R", Mapped: true, Candidates: c.candidates})
			if refused.ID != "" {
				t.Fatalf("refused: %+v", refused)
			}
			if got.Method != LinkByPosition || !sameFIDs(fids(got), c.want) {
				t.Fatalf("link = %+v, want %v", got, c.want)
			}
		})
	}
}

func TestALinkSaysWhereTheWaterEntersEachStretch(t *testing.T) {
	got, _ := link(t, ReservoirPlace{ID: "r", Name: "R", Mapped: true, Candidates: []ReservoirCandidate{inside(1, 50)}})
	stream, main := got.Reaches[0], got.Reaches[1]
	if !stream.Into || stream.RiverCMS != 1 || stream.RiverUplandKM2 != 50 {
		t.Fatalf("stream = %+v", stream)
	}
	if main.Into || main.RiverCMS != 35 || main.RiverUplandKM2 != 700 {
		t.Fatalf("main stem = %+v", main)
	}
}

func TestLinkByRiverOnlyWhenItLeavesOneStretch(t *testing.T) {
	cases := []struct {
		name         string
		river        *string
		district     *string
		want         []int64
		refuseReason string
	}{
		{"a river with one stretch", str("Stream Oya"), nil, []int64{100}, ""},
		{"a spelling the source uses", str("stream  oya"), nil, []int64{100}, ""},
		{"an alias", str("Old Name"), str("Ampara"), nil, UnlinkedAmbiguous},
		{"one stretch of the river in its district", str("Main Ganga"), str("Matale"), []int64{300}, ""},
		{"several stretches in its district", str("Main Ganga"), str("Kandy"), nil, UnlinkedAmbiguous},
		{"no district on a long river", str("Main Ganga"), nil, nil, UnlinkedAmbiguous},
		{"a river the network does not know", str("Nowhere Aru"), nil, nil, UnlinkedNoPlace},
		{"no river at all", nil, str("Matale"), nil, UnlinkedNoPlace},
		{"two rivers named, one known", str("Stream Oya & Nowhere Aru"), nil, []int64{100}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, refused := link(t, ReservoirPlace{ID: "r", Name: "R", River: c.river, District: c.district})
			if c.refuseReason != "" {
				if refused.Reason != c.refuseReason {
					t.Fatalf("got %+v %+v, want refused for %s", got, refused, c.refuseReason)
				}
				return
			}
			if got.Method != LinkByRiver || !sameFIDs(fids(got), c.want) {
				t.Fatalf("link = %+v %+v, want %v", got, refused, c.want)
			}
		})
	}
}

func TestAReservoirIsNeverGuessedOntoARiver(t *testing.T) {
	cases := []struct {
		name  string
		place ReservoirPlace
		want  string
	}{
		{"nothing within the limit", ReservoirPlace{Mapped: true, Candidates: []ReservoirCandidate{{SegmentID: 1, UplandKM2: 50, DistanceM: 1600}}}, UnlinkedTooFar},
		{"only a rill nearby", ReservoirPlace{Mapped: true, Candidates: []ReservoirCandidate{{SegmentID: 1, UplandKM2: 14, Inside: true}}}, UnlinkedTooFar},
		{"position and river disagree", ReservoirPlace{Mapped: true, River: str("Other Oya"), Candidates: []ReservoirCandidate{inside(1, 50)}}, UnlinkedConflict},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.place.ID, c.place.Name = "r", "R"
			if _, refused := link(t, c.place); refused.Reason != c.want {
				t.Fatalf("refused = %+v, want %s", refused, c.want)
			}
		})
	}
	// A river the network does not know says nothing against a position.
	got, _ := link(t, ReservoirPlace{ID: "r", Name: "R", Mapped: true, River: str("Nowhere Aru"), Candidates: []ReservoirCandidate{inside(1, 50)}})
	if got.Method != LinkByPosition {
		t.Fatalf("link = %+v", got)
	}
}

func TestAStreamThatReachesNoStretchIsSaidSo(t *testing.T) {
	segments := []LinkSegment{{ID: 1, NextDown: 0, MainRiv: 1, LengthKM: 5, UplandKM2: 40}}
	links, unlinked := LinkReservoirs([]ReservoirPlace{{ID: "r", Name: "R", Mapped: true, Candidates: []ReservoirCandidate{inside(1, 40)}}}, segments, nil, nil)
	if len(links) != 0 || len(unlinked) != 1 || unlinked[0].Reason != UnlinkedNoStretch {
		t.Fatalf("links %+v unlinked %+v", links, unlinked)
	}
}

func TestReachReservoirsCarryTheReadingToEachStretch(t *testing.T) {
	cusecs := 1551.0
	layer := reservoirs.FeatureCollection{Features: []reservoirs.Feature{{Properties: reservoirs.Properties{
		ID: "kimbulwana-oya", Name: "Kimbulwana Oya", Status: "spilling", Type: "major",
		Flow: &reservoirs.Flow{Kind: "spill", Cusecs: &cusecs},
	}}}}
	links := []ReservoirLink{
		{ID: "kimbulwana-oya", Method: LinkByPosition, Reaches: []LinkedReach{{FID: 1, Into: true, RiverCMS: 2.9}, {FID: 2, RiverCMS: 33}}},
		{ID: "gone", Method: LinkByPosition, Reaches: []LinkedReach{{FID: 1}}},
	}
	got := reachReservoirs(links, layer)
	if len(got[1]) != 1 || !got[1][0].Into || *got[1][0].Cusecs != 1551 || got[1][0].FlowKind != "spill" {
		t.Fatalf("first = %+v", got[1])
	}
	if len(got[2]) != 1 || got[2][0].Into || got[2][0].RiverCMS != 33 {
		t.Fatalf("second = %+v", got[2])
	}
}

func TestAssembleShowsASpillInTheStretchAndTheRiver(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	layer := rivers.FeatureCollection{Features: []rivers.Feature{
		stationFeature("hanwella", 2, rivers.StatusNormal, false, now.Add(-time.Hour)),
	}}
	cusecs := 1551.0
	spills := map[int64][]Reservoir{4: {{Name: "Kimbulwana Oya", Status: "spilling", FlowKind: "spill", Cusecs: &cusecs, Into: true}}}
	result := assemble(testNetwork(), layer, []floods.Gauge{}, nil, rainMeta{}, spills, now, false)
	river := result.river["river-9"]
	if river.Level != LevelWatch || river.Reasons[0].Kind != "reservoir" {
		t.Fatalf("river = %+v", river)
	}
	if result.risk.Rivers[0].ID != "river-9" {
		t.Fatalf("the river at watch should lead: %v", result.risk.Rivers[0].ID)
	}
}
