package environment

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/prabhavalabs/lanka-weather/internal/grid"
)

func TestAQICategoryUsesTheEPABands(t *testing.T) {
	for index, want := range map[float64]string{
		0: AQIGood, 50: AQIGood, 51: AQIModerate, 100: AQIModerate, 101: AQIUnhealthySensitive, 150: AQIUnhealthySensitive,
		151: AQIUnhealthy, 200: AQIUnhealthy, 201: AQIVeryUnhealthy, 300: AQIVeryUnhealthy, 301: AQIHazardous,
	} {
		if got := AQICategory(index); got != want {
			t.Errorf("AQI %v = %s, want %s", index, got, want)
		}
	}
}

func TestSeaStateFollowsWMOHeights(t *testing.T) {
	for height, want := range map[float64]string{
		0: "calm", 0.3: "smooth", 1: "slight", 2: "moderate", 3: "rough", 5: "very rough", 8: "high", 12: "very high",
	} {
		if got := SeaState(height); got != want {
			t.Errorf("%v m = %s, want %s", height, got, want)
		}
	}
}

func TestBuildGridWritesMissingNodesAsNullAndCarriesTheLand(t *testing.T) {
	sea := grid.Sea()
	values := make([]float32, sea.Size())
	for index := range values {
		values[index] = float32(math.NaN())
	}
	values[sea.Nodes[0].Index] = 1.26
	current := &run{source: "s", model: "m", sourceTime: time.Unix(1793648700, 0).UTC(), fetchedAt: time.Unix(1793648800, 0).UTC(),
		lattice: sea.Lattice, gridID: sea.ID, fields: map[string][]float32{"wave_height": values}}
	body, err := json.Marshal(BuildGrid(current, seaVariables, "credit", seaNote, grid.Land()))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{`"waveHeightM":[`, `null`, `1.3`, `"land":[[[`, `"credit"`, `sea-area warnings`, `"m"`} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is missing from %s", want, text[:min(300, len(text))])
		}
	}
	if strings.Contains(text, "NaN") || strings.Contains(text, "swellHeightM") {
		t.Fatalf("unexpected content: %s", text[:300])
	}
}

func TestKnownGridsAreRegistered(t *testing.T) {
	if Grids[grid.AirQualityID].Size() == 0 || Grids[grid.SeaID].Size() == 0 {
		t.Fatal("grids are not registered")
	}
}
