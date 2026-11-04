package rivernet

import "testing"

func TestChooseStationSegment(t *testing.T) {
	const kelani, other = int64(1), int64(2)
	stem := map[int64]bool{91: true}
	cases := []struct {
		name       string
		candidates []Candidate
		hint       StationHint
		want       int64
		method     string
		ok         bool
	}{
		{"on the main stem if the department says so, however near a canal runs", []Candidate{
			{SegmentID: 90, MainRiv: kelani, UplandKM2: 96, DistanceM: 37},
			{SegmentID: 91, MainRiv: kelani, UplandKM2: 2300, DistanceM: 2100},
		}, StationHint{System: kelani, MainStem: stem}, 91, "main-stem", true},
		{"nearest when the main stem is out of reach", []Candidate{
			{SegmentID: 92, MainRiv: kelani, UplandKM2: 96, DistanceM: 37},
		}, StationHint{System: kelani, MainStem: stem}, 92, "basin", true},
		{"nearest in its own system", []Candidate{
			{SegmentID: 10, MainRiv: kelani, UplandKM2: 500, DistanceM: 200},
			{SegmentID: 11, MainRiv: kelani, UplandKM2: 500, DistanceM: 1500},
		}, StationHint{System: kelani}, 10, "basin", true},
		{"the larger river wins a near tie: a rill runs closer to the gauge", []Candidate{
			{SegmentID: 20, MainRiv: kelani, UplandKM2: 20, DistanceM: 100},
			{SegmentID: 21, MainRiv: kelani, UplandKM2: 2000, DistanceM: 400},
		}, StationHint{System: kelani}, 21, "basin", true},
		{"but not when clearly further", []Candidate{
			{SegmentID: 30, MainRiv: kelani, UplandKM2: 40, DistanceM: 100},
			{SegmentID: 31, MainRiv: kelani, UplandKM2: 2000, DistanceM: 1500},
		}, StationHint{System: kelani}, 30, "basin", true},
		{"its own system beats a nearer other one", []Candidate{
			{SegmentID: 40, MainRiv: other, UplandKM2: 900, DistanceM: 100},
			{SegmentID: 41, MainRiv: kelani, UplandKM2: 300, DistanceM: 2000},
		}, StationHint{System: kelani}, 41, "basin", true},
		{"a tiny stream is not suitable", []Candidate{
			{SegmentID: 50, MainRiv: kelani, UplandKM2: 5, DistanceM: 50},
		}, StationHint{System: kelani}, 0, "", false},
		{"too far away", []Candidate{
			{SegmentID: 60, MainRiv: kelani, UplandKM2: 500, DistanceM: 3001},
		}, StationHint{System: kelani}, 0, "", false},
		{"an unknown basin falls back to position", []Candidate{
			{SegmentID: 70, MainRiv: other, UplandKM2: 500, DistanceM: 100},
		}, StationHint{}, 70, "position", true},
		{"a basin not found nearby falls back to position", []Candidate{
			{SegmentID: 80, MainRiv: other, UplandKM2: 500, DistanceM: 100},
		}, StationHint{System: kelani}, 80, "position", true},
		{"nothing at all", nil, StationHint{System: kelani}, 0, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, method, ok := ChooseStationSegment(c.candidates, c.hint)
			if ok != c.ok || got.SegmentID != c.want || method != c.method {
				t.Fatalf("got %d %q %v, want %d %q %v", got.SegmentID, method, ok, c.want, c.method, c.ok)
			}
		})
	}
}

func TestChooseGoogleSegment(t *testing.T) {
	cases := []struct {
		name       string
		candidates []Candidate
		hybas      int64
		want       int64
		method     string
		ok         bool
	}{
		{"the river leaving the sub-basin", []Candidate{
			{SegmentID: 1, HybasL12: 9, UplandKM2: 30, DistanceM: 100},
			{SegmentID: 2, HybasL12: 9, UplandKM2: 800, DistanceM: 900},
			{SegmentID: 3, HybasL12: 8, UplandKM2: 2000, DistanceM: 50},
		}, 9, 2, "hybas", true},
		{"proximity without a matching sub-basin", []Candidate{
			{SegmentID: 4, HybasL12: 8, UplandKM2: 300, DistanceM: 500},
		}, 9, 4, "position", true},
		{"a sub-basin match still has a distance limit", []Candidate{
			{SegmentID: 5, HybasL12: 9, UplandKM2: 300, DistanceM: 9000},
		}, 9, 0, "", false},
		{"a gauge that is not a hybas id", []Candidate{
			{SegmentID: 6, HybasL12: 8, UplandKM2: 300, DistanceM: 100},
		}, 0, 6, "position", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, method, ok := ChooseGoogleSegment(c.candidates, c.hybas)
			if ok != c.ok || got.SegmentID != c.want || method != c.method {
				t.Fatalf("got %d %q %v", got.SegmentID, method, ok)
			}
		})
	}
}

func TestHybasOfGauge(t *testing.T) {
	if got := HybasOfGauge("hybas_4121599890"); got != 4121599890 {
		t.Fatalf("got %d", got)
	}
	for _, id := range []string{"", "gauge_1", "hybas_x"} {
		if got := HybasOfGauge(id); got != 0 {
			t.Fatalf("%q = %d", id, got)
		}
	}
}

func TestTributariesFromVotes(t *testing.T) {
	segments := []Segment{
		{ID: 1, NextDown: 2, MainRiv: 9, LengthKM: 5, UplandKM2: 30},
		{ID: 2, NextDown: 3, MainRiv: 9, LengthKM: 5, UplandKM2: 60},
		{ID: 3, NextDown: 4, MainRiv: 9, LengthKM: 5, UplandKM2: 120},
		{ID: 4, NextDown: 9, MainRiv: 9, LengthKM: 5, UplandKM2: 200},
		{ID: 9, NextDown: 0, MainRiv: 9, LengthKM: 5, UplandKM2: 2000},
		{ID: 5, NextDown: 9, MainRiv: 9, LengthKM: 3, UplandKM2: 150},
	}
	got := TributariesFromVotes(segments, []StreamVote{
		{SegmentID: 2, Name: "Uma Oya"},
		{SegmentID: 3, Name: "Uma Oya"},
		{SegmentID: 4, Name: "Uma Oya"},
		// The river's own name says nothing new.
		{SegmentID: 9, Name: "Kelani Ganga"},
		// Too short a chain to trust.
		{SegmentID: 5, Name: "Short Oya"},
	}, map[int64]string{9: "Kelani Ganga"})
	if len(got) != 1 || got[0].Name != "Uma Oya" {
		t.Fatalf("tributaries = %+v", got)
	}
	// Carried on up the largest branch to the top of the stem, and stopping at
	// the last segment the map names.
	if got[0].SourceID != 1 || got[0].EndID != 4 {
		t.Fatalf("source = %d, end = %d", got[0].SourceID, got[0].EndID)
	}
}

func TestTributariesFromVotesKeepsTheLongestChainOfASharedName(t *testing.T) {
	segments := []Segment{
		{ID: 1, NextDown: 2, MainRiv: 9, LengthKM: 6, UplandKM2: 150},
		{ID: 2, NextDown: 3, MainRiv: 9, LengthKM: 6, UplandKM2: 160},
		{ID: 3, NextDown: 9, MainRiv: 9, LengthKM: 1, UplandKM2: 400},
		{ID: 4, NextDown: 9, MainRiv: 9, LengthKM: 2, UplandKM2: 120},
		{ID: 9, NextDown: 0, MainRiv: 9, LengthKM: 5, UplandKM2: 2000},
	}
	got := TributariesFromVotes(segments, []StreamVote{
		{SegmentID: 1, Name: "Kuda Oya"}, {SegmentID: 2, Name: "Kuda Oya"}, {SegmentID: 4, Name: "Kuda Oya"},
	}, nil)
	if len(got) != 1 || got[0].SourceID != 1 {
		t.Fatalf("tributaries = %+v", got)
	}
}

func TestSourceAboveStopsAtSmallStreams(t *testing.T) {
	segments := []Segment{
		{ID: 1, NextDown: 2, UplandKM2: 10},
		{ID: 2, NextDown: 3, UplandKM2: 30},
		{ID: 5, NextDown: 3, UplandKM2: 28},
		{ID: 3, NextDown: 0, UplandKM2: 90},
	}
	if got := SourceAbove(segments, 3); got != 2 {
		t.Fatalf("source = %d", got)
	}
	if got := SourceAbove(segments, 1); got != 1 {
		t.Fatalf("a headwater is its own source, got %d", got)
	}
}

func TestMainStemsFollowTheLargestBranch(t *testing.T) {
	stems := MainStems([]Segment{
		{ID: 1, NextDown: 3, MainRiv: 3, UplandKM2: 40},
		{ID: 2, NextDown: 3, MainRiv: 3, UplandKM2: 30},
		{ID: 4, NextDown: 1, MainRiv: 3, UplandKM2: 10},
		{ID: 3, NextDown: 0, MainRiv: 3, UplandKM2: 90},
	})
	stem := stems[3]
	if !stem[3] || !stem[1] || stem[2] || stem[4] {
		t.Fatalf("stem = %v", stem)
	}
}
