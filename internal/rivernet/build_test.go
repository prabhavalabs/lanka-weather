package rivernet

import (
	"slices"
	"testing"
)

// A river system drawn top to bottom, with ids by position:
//
//	1 (40)      2 (30)
//	  \          /
//	   3 (100)      8 (60)  <- a tributary
//	     \            /
//	      4 (200)    /
//	        \       /
//	         5 (300)
//	           |
//	         6 (400)
//	           |
//	         7 (500) sea
//
// and a small separate system, 20 → 21.
func fixture() []Segment {
	return []Segment{
		{ID: 1, NextDown: 3, MainRiv: 7, LengthKM: 5, UplandKM2: 40},
		{ID: 2, NextDown: 3, MainRiv: 7, LengthKM: 5, UplandKM2: 30},
		{ID: 3, NextDown: 4, MainRiv: 7, LengthKM: 5, UplandKM2: 100},
		{ID: 4, NextDown: 5, MainRiv: 7, LengthKM: 5, UplandKM2: 200},
		{ID: 5, NextDown: 6, MainRiv: 7, LengthKM: 5, UplandKM2: 300},
		{ID: 6, NextDown: 7, MainRiv: 7, LengthKM: 5, UplandKM2: 400},
		{ID: 7, NextDown: 0, MainRiv: 7, LengthKM: 5, UplandKM2: 500},
		{ID: 8, NextDown: 5, MainRiv: 7, LengthKM: 5, UplandKM2: 60},
		{ID: 20, NextDown: 21, MainRiv: 21, LengthKM: 2, UplandKM2: 10},
		{ID: 21, NextDown: 0, MainRiv: 21, LengthKM: 2, UplandKM2: 20},
	}
}

func named(segments []Segment, gauges ...Gauge) Inputs {
	return Inputs{
		Segments:    segments,
		SystemNames: map[int64]SystemName{7: {Name: "Kelani Ganga", BasinFID: 580}},
		Gauges:      gauges,
	}
}

func riverByID(t *testing.T, network Network, id string) River {
	t.Helper()
	for _, river := range network.Rivers {
		if river.ID == id {
			return river
		}
	}
	t.Fatalf("no river %q in %+v", id, network.Rivers)
	return River{}
}

func reachesOf(network Network, id string) []Reach {
	var out []Reach
	for _, reach := range network.Reaches {
		if reach.RiverID == id {
			out = append(out, reach)
		}
	}
	return out
}

func gaugeIDs(reach Reach) []string {
	var ids []string
	for _, ref := range reach.Upstream {
		ids = append(ids, ref.ID)
	}
	return ids
}

func TestMainStemFollowsTheLargestBranchToTheSea(t *testing.T) {
	network := Build(named(fixture()))
	river := riverByID(t, network, "kelani-ganga")
	if river.MouthID != 7 || !river.Named || river.BasinFID != 580 || river.NameSI == "" {
		t.Fatalf("river = %+v", river)
	}
	for _, id := range []int64{1, 3, 4, 5, 6, 7} {
		if network.SegmentRiver[id] != "kelani-ganga" {
			t.Fatalf("segment %d is on %q", id, network.SegmentRiver[id])
		}
	}
	for _, id := range []int64{2, 8} {
		if _, ok := network.SegmentRiver[id]; ok {
			t.Fatalf("branch %d should stay a stream", id)
		}
	}
	if river.LengthKM != 30 {
		t.Fatalf("length = %v", river.LengthKM)
	}
}

func TestAnUngaugedRiverIsOneHonestReach(t *testing.T) {
	network := Build(named(fixture()))
	reaches := reachesOf(network, "kelani-ganga")
	if len(reaches) != 1 || reaches[0].Gauged() || reaches[0].Name != "Whole river" || reaches[0].FID != 7 {
		t.Fatalf("reaches = %+v", reaches)
	}
}

func TestGaugesSplitAStemIntoReachesWalkingDownstream(t *testing.T) {
	network := Build(named(fixture(),
		Gauge{Kind: "station", ID: "upper", Label: "Upper", SegmentID: 4},
		Gauge{Kind: "station", ID: "lower", Label: "Lower", SegmentID: 6},
	))
	reaches := reachesOf(network, "kelani-ganga")
	if len(reaches) != 3 {
		t.Fatalf("reaches = %+v", reaches)
	}
	above, between, below := reaches[0], reaches[1], reaches[2]
	if above.Gauged() || above.Next == nil || above.Next.ID != "upper" || above.Name != "Above Upper" {
		t.Fatalf("above = %+v", above)
	}
	if !slices.Equal(gaugeIDs(between), []string{"upper"}) || between.Downstream == nil || between.Downstream.ID != "lower" || between.Name != "Upper to Lower" {
		t.Fatalf("between = %+v", between)
	}
	if !slices.Equal(gaugeIDs(below), []string{"lower"}) || below.Downstream != nil || below.Name != "Below Lower" || below.FID != 7 {
		t.Fatalf("below = %+v", below)
	}
	if above.Position != 0 || between.Position != 1 || below.Position != 2 {
		t.Fatalf("positions %d %d %d", above.Position, between.Position, below.Position)
	}
	// Every stem segment lands in exactly one reach.
	for id, want := range map[int64]int64{1: above.FID, 3: above.FID, 4: between.FID, 5: between.FID, 6: below.FID, 7: below.FID} {
		if network.SegmentReach[id] != want {
			t.Fatalf("segment %d in reach %d, want %d", id, network.SegmentReach[id], want)
		}
	}
}

func TestAGaugeOnAnUnnamedTributaryGetsAStemToTheRiver(t *testing.T) {
	network := Build(named(fixture(), Gauge{Kind: "station", ID: "trib", Label: "Trib", SegmentID: 8}))
	if network.SegmentRiver[8] != "kelani-ganga" {
		t.Fatalf("the gauged tributary should join its river, got %q", network.SegmentRiver[8])
	}
	var tributary *Reach
	for _, reach := range network.Reaches {
		if reach.FID == 8 {
			tributary = &reach
		}
	}
	if tributary == nil || !tributary.Gauged() {
		t.Fatalf("reaches = %+v", network.Reaches)
	}
}

func TestAMainBranchGaugeBeatsATributaryGaugeBelowTheConfluence(t *testing.T) {
	network := Build(named(fixture(),
		Gauge{Kind: "station", ID: "main", Label: "Main", SegmentID: 4},
		Gauge{Kind: "station", ID: "trib", Label: "Trib", SegmentID: 8},
	))
	for _, reach := range network.Reaches {
		if reach.FID == 7 && !slices.Equal(gaugeIDs(reach), []string{"main"}) {
			t.Fatalf("below the confluence = %+v", reach)
		}
	}
}

func TestATributaryGaugeGovernsAMainStemWithNoGaugeOfItsOwn(t *testing.T) {
	network := Build(named(fixture(), Gauge{Kind: "station", ID: "trib", Label: "Trib", SegmentID: 8}))
	for _, reach := range network.Reaches {
		if reach.FID == 7 && !slices.Equal(gaugeIDs(reach), []string{"trib"}) {
			t.Fatalf("below the confluence = %+v", reach)
		}
	}
}

func TestTwoGaugesOnOneSegmentBothGovern(t *testing.T) {
	network := Build(named(fixture(),
		Gauge{Kind: "station", ID: "a", Label: "A", SegmentID: 6},
		Gauge{Kind: "google", ID: "b", Label: "B", SegmentID: 6},
	))
	reaches := reachesOf(network, "kelani-ganga")
	if got := gaugeIDs(reaches[len(reaches)-1]); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("gauges = %v", got)
	}
}

func TestNamedTributaries(t *testing.T) {
	inputs := named(fixture())
	// Segment 2 names a tributary from its source.
	inputs.SourceNames = map[int64]string{2: "We Oya", 7: "Kelani Ganga"}
	network := Build(inputs)
	tributary := riverByID(t, network, "we-oya")
	if tributary.ParentID != "kelani-ganga" || tributary.MouthID != 2 {
		t.Fatalf("tributary = %+v", tributary)
	}
	if network.SegmentRiver[2] != "we-oya" {
		t.Fatalf("segment 2 on %q", network.SegmentRiver[2])
	}
}

func TestABranchWithTheRiversOwnNameExtendsTheRiver(t *testing.T) {
	inputs := named(fixture())
	inputs.SourceNames = map[int64]string{8: "Kelani Ganga"}
	network := Build(inputs)
	if network.SegmentRiver[8] != "kelani-ganga" {
		t.Fatalf("segment 8 on %q", network.SegmentRiver[8])
	}
	for _, river := range network.Rivers {
		if river.ParentID != "" {
			t.Fatalf("no tributary should be made: %+v", river)
		}
	}
}

func TestAHeadwaterMisnamedAfterTheMainRiverIsNotMadeARiver(t *testing.T) {
	// 10 runs into the tributary 8, not the main river. lk_rivers gives it the
	// main river's name, which cannot be right there.
	segments := append(fixture(), Segment{ID: 10, NextDown: 8, MainRiv: 7, LengthKM: 4, UplandKM2: 30})
	inputs := named(segments)
	inputs.Tributaries = []Tributary{{SourceID: 8, Name: "Gurugoda Oya"}}
	inputs.SourceNames = map[int64]string{10: "Kelani Ganga"}
	network := Build(inputs)
	if got := network.SegmentRiver[8]; got != "gurugoda-oya" {
		t.Fatalf("segment 8 on %q", got)
	}
	if got := network.SegmentRiver[10]; got == "kelani-ganga" {
		t.Fatalf("the headwater was put on the main river")
	}
}

func TestTwoBranchesOfOneNameAreOneRiver(t *testing.T) {
	segments := append(fixture(),
		Segment{ID: 10, NextDown: 5, MainRiv: 7, LengthKM: 4, UplandKM2: 60},
	)
	inputs := named(segments)
	inputs.Tributaries = []Tributary{{SourceID: 8, Name: "Gurugoda Oya"}, {SourceID: 10, Name: "Gurugoda Oya"}}
	network := Build(inputs)
	if network.SegmentRiver[8] != "gurugoda-oya" || network.SegmentRiver[10] != "gurugoda-oya" {
		t.Fatalf("assignments = %v", network.SegmentRiver)
	}
	for _, river := range network.Rivers {
		if river.ID != "gurugoda-oya" && river.Name == "Gurugoda Oya" {
			t.Fatalf("a second river of the same name: %+v", river)
		}
	}
}

func TestANamedHeadwaterTooSmallToDrawIsDropped(t *testing.T) {
	segments := append(fixture(), Segment{ID: 10, NextDown: 5, MainRiv: 7, LengthKM: 3, UplandKM2: 6})
	inputs := named(segments)
	inputs.SourceNames = map[int64]string{10: "Kehelgamu Oya"}
	network := Build(inputs)
	if _, ok := network.SegmentRiver[10]; ok {
		t.Fatalf("assignments = %v", network.SegmentRiver)
	}
}

func TestAReadNameStopsWhereTheMapStopsAndLargerRiversKeepTheirCourse(t *testing.T) {
	// 10 → 11 → 12 → 5. The map names 12 "Amban Ganga" and 10 "Kambarawa Ganga",
	// whose chain would otherwise run down over 12.
	segments := append(fixture(),
		Segment{ID: 10, NextDown: 11, MainRiv: 7, LengthKM: 5, UplandKM2: 30},
		Segment{ID: 11, NextDown: 12, MainRiv: 7, LengthKM: 5, UplandKM2: 50},
		Segment{ID: 12, NextDown: 5, MainRiv: 7, LengthKM: 5, UplandKM2: 90},
	)
	inputs := named(segments)
	inputs.Tributaries = []Tributary{
		{SourceID: 10, EndID: 10, Name: "Kambarawa Ganga"},
		{SourceID: 12, EndID: 12, Name: "Amban Ganga"},
	}
	network := Build(inputs)
	if network.SegmentRiver[12] != "amban-ganga" {
		t.Fatalf("segment 12 on %q", network.SegmentRiver[12])
	}
	if network.SegmentRiver[10] != "kambarawa-ganga" {
		t.Fatalf("segment 10 on %q", network.SegmentRiver[10])
	}
	// The unnamed gap joins the name above it, and the tree is whole.
	if network.SegmentRiver[11] != "kambarawa-ganga" {
		t.Fatalf("segment 11 on %q", network.SegmentRiver[11])
	}
	if river := riverByID(t, network, "kambarawa-ganga"); river.ParentID != "amban-ganga" {
		t.Fatalf("parent = %q", river.ParentID)
	}
	if river := riverByID(t, network, "amban-ganga"); river.ParentID != "kelani-ganga" {
		t.Fatalf("parent = %q", river.ParentID)
	}
}

func TestTributariesSharingACourseSplitItStably(t *testing.T) {
	// 10 and 11 join at 12, which joins the river at 5. One of them takes 12
	// and the other stops at the junction; each keeps its own headwater.
	segments := append(fixture(),
		Segment{ID: 10, NextDown: 12, MainRiv: 7, UplandKM2: 30},
		Segment{ID: 11, NextDown: 12, MainRiv: 7, UplandKM2: 50},
		Segment{ID: 12, NextDown: 5, MainRiv: 7, UplandKM2: 90},
	)
	inputs := named(segments)
	inputs.Tributaries = []Tributary{{SourceID: 10, Name: "Small Oya"}, {SourceID: 11, Name: "Big Oya"}}
	network := Build(inputs)
	if network.SegmentRiver[12] == "" {
		t.Fatalf("segment 12 unassigned: %v", network.SegmentRiver)
	}
	if network.SegmentRiver[10] != "small-oya" || network.SegmentRiver[11] != "big-oya" {
		t.Fatalf("assignments = %v", network.SegmentRiver)
	}
}

func TestSmallUnnamedSystemsAreLeftAsStreamsUnlessGauged(t *testing.T) {
	network := Build(named(fixture()))
	if _, ok := network.SegmentRiver[21]; ok {
		t.Fatal("a small unnamed system should not be a river")
	}
	gauged := Build(named(fixture(), Gauge{Kind: "google", ID: "g", Label: "G", SegmentID: 21}))
	if gauged.SegmentRiver[21] == "" {
		t.Fatal("a gauged system must have a river, or its gauge governs nothing")
	}
	river := riverByID(t, gauged, gauged.SegmentRiver[21])
	if river.Named || river.Name != "" {
		t.Fatalf("river = %+v", river)
	}
}

func TestLargeUnnamedSystemsBecomeRivers(t *testing.T) {
	segments := []Segment{{ID: 30, NextDown: 0, MainRiv: 30, UplandKM2: 150}}
	network := Build(Inputs{Segments: segments})
	if len(network.Rivers) != 1 || network.Rivers[0].Named || network.Rivers[0].ID != "river-30" {
		t.Fatalf("rivers = %+v", network.Rivers)
	}
}

func TestAnOutletKeepsLkRiversNameWhenTheBasinsGiveNone(t *testing.T) {
	segments := []Segment{{ID: 30, NextDown: 0, MainRiv: 30, UplandKM2: 150}}
	network := Build(Inputs{Segments: segments, SourceNames: map[int64]string{30: "Some Aru"}})
	if network.Rivers[0].Name != "Some Aru" || len(network.Rivers[0].Aliases) != 0 {
		t.Fatalf("river = %+v", network.Rivers[0])
	}
}

func TestDisplayNamesKeepTheOfficialNameAsAnAlias(t *testing.T) {
	segments := []Segment{{ID: 30, NextDown: 0, MainRiv: 30, UplandKM2: 3000}}
	network := Build(Inputs{
		Segments:    segments,
		SystemNames: map[int64]SystemName{30: {Name: "Aruvi Aru", BasinFID: 1}},
		SourceNames: map[int64]string{30: "Malwathu Oya"},
	})
	river := network.Rivers[0]
	if river.Name != "Malwathu Oya" || river.ID != "malwathu-oya" || !slices.Contains(river.Aliases, "Aruvi Aru") {
		t.Fatalf("river = %+v", river)
	}
}

func TestGaugesOffTheNetworkAreIgnored(t *testing.T) {
	network := Build(named(fixture(), Gauge{Kind: "station", ID: "lost", Label: "Lost", SegmentID: 999}))
	if reaches := reachesOf(network, "kelani-ganga"); len(reaches) != 1 || reaches[0].Gauged() {
		t.Fatalf("reaches = %+v", reaches)
	}
}

func TestResolveSystemNames(t *testing.T) {
	names := ResolveSystemNames([]BasinVote{
		{MainRiv: 1, BasinFID: 10, Basin: "Kala Oya", CatchKM2: 2500},
		{MainRiv: 1, BasinFID: 11, Basin: "Mi Oya", CatchKM2: 20},
		// A small coastal stream beside the mouth: same basin, different river.
		{MainRiv: 2, BasinFID: 10, Basin: "Kala Oya", CatchKM2: 30},
		// A system split evenly names itself after neither unless one holds half.
		{MainRiv: 3, BasinFID: 12, Basin: "A Aru", CatchKM2: 40},
		{MainRiv: 3, BasinFID: 13, Basin: "B Aru", CatchKM2: 30},
		{MainRiv: 3, BasinFID: 14, Basin: "C Aru", CatchKM2: 30},
	})
	if names[1].Name != "Kala Oya" || names[1].BasinFID != 10 {
		t.Fatalf("names = %+v", names)
	}
	if _, ok := names[2]; ok {
		t.Fatalf("the small stream took the basin's name: %+v", names)
	}
	if _, ok := names[3]; ok {
		t.Fatalf("a system no basin holds half of got a name: %+v", names)
	}
}

func TestCanonicalBasin(t *testing.T) {
	for raw, want := range map[string]string{
		"Mee Oya":             "mi oya",
		"Aththanagalu Oya":    "attanagalu oya",
		"  Kelani   Ganga  ":  "kelani ganga",
		"A basin no one knew": "a basin no one knew",
	} {
		if got := CanonicalBasin(raw); got != want {
			t.Fatalf("%q = %q, want %q", raw, got, want)
		}
	}
}
