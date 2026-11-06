package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/floodrisk"
)

type fakeFloodRisk struct{}

func (fakeFloodRisk) Estimate(context.Context) (*floodrisk.Document, error) {
	inputs := floodrisk.Inputs{District: "LK-11", FlashLevel: 1, Rain: &floodrisk.ObservedRain{Rise1hMm: 45, Gauge: "Kesbewa"}}
	return &floodrisk.Document{Districts: []floodrisk.District{{Assessment: floodrisk.Assess(inputs), Inputs: inputs}},
		Meta: floodrisk.Meta{Note: floodrisk.Note, Complete: true, Missing: []string{}}}, nil
}

func TestFloodRiskServesEachDistrictWithItsReasons(t *testing.T) {
	server := newServer(t, Dependencies{FloodRisk: fakeFloodRisk{}})
	resp, body := get(t, server, "/v1/flood-risk?district=lk-11")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	var envelope struct {
		Data struct {
			Districts []struct {
				District string `json:"district"`
				Level    int    `json:"level"`
				Variant  string `json:"variant"`
				Signals  []struct {
					Family string `json:"family"`
				} `json:"signals"`
				Inputs struct {
					Rain struct {
						Gauge string `json:"gauge"`
					} `json:"rain"`
				} `json:"inputs"`
			} `json:"districts"`
			Meta struct {
				Note string `json:"note"`
			} `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	districts := envelope.Data.Districts
	if len(districts) != 1 || districts[0].Variant != "high" || len(districts[0].Signals) != 2 ||
		districts[0].Inputs.Rain.Gauge != "Kesbewa" || envelope.Data.Meta.Note == "" {
		t.Fatalf("data %s", body)
	}
	// A district nothing is known of is level 0, not missing.
	_, body = get(t, server, "/v1/flood-risk?district=LK-92")
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Data.Districts) != 1 || envelope.Data.Districts[0].Level != 0 {
		t.Fatalf("LK-92 %s", body)
	}
	if resp, _ := get(t, server, "/v1/flood-risk?district=Colombo"); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
