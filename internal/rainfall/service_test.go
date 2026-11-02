package rainfall

import (
	"bytes"
	"image/png"
	"testing"
	"time"
)

func TestBuildStepsOrdersAndAgesTheNewestImage(t *testing.T) {
	first := time.Date(2026, 11, 2, 13, 0, 0, 0, time.UTC)
	steps := []Step{
		{Time: "b", ValidAt: first.Add(30 * time.Minute), EndsAt: first.Add(time.Hour), FetchedAt: first},
		{Time: "a", ValidAt: first, EndsAt: first.Add(30 * time.Minute), FetchedAt: first},
	}
	fresh := BuildSteps(steps, first.Add(7*time.Hour), 10*time.Hour)
	if fresh.Steps[0].Time != "a" || fresh.Stale || !fresh.NewestValidAt.Equal(first.Add(30*time.Minute)) || !fresh.Estimate {
		t.Fatalf("fresh = %+v", fresh)
	}
	if !BuildSteps(steps, first.Add(12*time.Hour), 10*time.Hour).Stale {
		t.Fatal("an image older than the limit should be stale")
	}
	none := BuildSteps(nil, first, 10*time.Hour)
	if !none.Stale || none.Steps == nil || none.NewestValidAt != nil {
		t.Fatalf("empty = %+v", none)
	}
	if none.Legend.Unit != "mm/h" || len(none.Legend.Stops) == 0 {
		t.Fatalf("legend = %+v", none.Legend)
	}
}

func TestEmptyTileIsATransparentPNG(t *testing.T) {
	decoded, err := png.Decode(bytes.NewReader(EmptyTile()))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, alpha := decoded.At(0, 0).RGBA(); alpha != 0 {
		t.Fatalf("alpha = %d", alpha)
	}
}
