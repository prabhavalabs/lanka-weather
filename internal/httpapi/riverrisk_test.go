package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/riverrisk"
)

type fakeRiverRisk struct {
	empty bool
	err   error
}

func (fake *fakeRiverRisk) Risk(context.Context) (*riverrisk.RiskDocument, error) {
	if fake.err != nil {
		return nil, fake.err
	}
	document := &riverrisk.RiskDocument{Estimate: true, Note: riverrisk.Note, Rivers: []riverrisk.RiverRisk{}}
	if !fake.empty {
		document.Rivers = append(document.Rivers, riverrisk.RiverRisk{ID: "kelani-ganga", Name: "Kelani Ganga", Level: riverrisk.LevelWatch,
			Reaches: []riverrisk.ReachRisk{{FID: 41405515, RiverID: "kelani-ganga", Name: "Below Nagalagam Street", Level: riverrisk.LevelWatch}}})
	}
	return document, nil
}

func (fake *fakeRiverRisk) River(_ context.Context, id string) (*riverrisk.RiverDetail, error) {
	if id != "kelani-ganga" {
		return nil, riverrisk.ErrNotFound
	}
	return &riverrisk.RiverDetail{RiverRisk: riverrisk.RiverRisk{ID: id, Name: "Kelani Ganga", Level: riverrisk.LevelWatch}, Estimate: true}, nil
}

func (fake *fakeRiverRisk) Basins(context.Context) (*riverrisk.BasinsDocument, error) {
	document := &riverrisk.BasinsDocument{Basins: []riverrisk.BasinStatus{}, Subbasins: []riverrisk.SubbasinStatus{}}
	if !fake.empty {
		document.Basins = append(document.Basins, riverrisk.BasinStatus{FID: 580, ID: "kelani-ganga", Name: "Kelani Ganga", State: riverrisk.BasinNoFlooding})
	}
	return document, nil
}

func (fake *fakeRiverRisk) Basin(_ context.Context, id string) (*riverrisk.BasinStatus, *riverrisk.BasinsMeta, error) {
	if id != "kelani-ganga" {
		return nil, nil, riverrisk.ErrNotFound
	}
	return &riverrisk.BasinStatus{FID: 580, ID: id, State: riverrisk.BasinNoFlooding}, &riverrisk.BasinsMeta{}, nil
}

func (fake *fakeRiverRisk) PastFloods(context.Context) (*riverrisk.PastFloodsDocument, error) {
	return &riverrisk.PastFloodsDocument{Note: riverrisk.PastFloodNote, Events: []riverrisk.PastFlood{{ID: "kelani-ganga-flood-2016", Year: 2016}}}, nil
}

func (fake *fakeRiverRisk) tile(z, _, _ int) (regions.Document, error) {
	if z == 3 {
		return regions.Document{ETag: `W/"v"`}, nil
	}
	return regions.Document{Body: []byte("tile"), ETag: `W/"v"`}, nil
}

func (fake *fakeRiverRisk) RiverTile(_ context.Context, z, x, y int) (regions.Document, error) {
	return fake.tile(z, x, y)
}

func (fake *fakeRiverRisk) BasinTile(_ context.Context, z, x, y int) (regions.Document, error) {
	return fake.tile(z, x, y)
}

func (fake *fakeRiverRisk) PastFloodTile(_ context.Context, z, x, y int) (regions.Document, error) {
	return fake.tile(z, x, y)
}

func TestRiverRiskListsRiversAndSaysItIsAnEstimate(t *testing.T) {
	server := newServer(t, Dependencies{RiverRisk: &fakeRiverRisk{}})
	resp, body := get(t, server, "/v1/rivers/risk")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	var envelope struct {
		Data struct {
			Estimate bool   `json:"estimate"`
			Note     string `json:"note"`
			Rivers   []struct {
				ID      string `json:"id"`
				Level   string `json:"level"`
				Reaches []struct {
					FID int64 `json:"fid"`
				} `json:"reaches"`
			} `json:"rivers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Data.Estimate || envelope.Data.Note == "" {
		t.Fatalf("not labelled an estimate: %s", body)
	}
	if len(envelope.Data.Rivers) != 1 || envelope.Data.Rivers[0].Level != "watch" || envelope.Data.Rivers[0].Reaches[0].FID != 41405515 {
		t.Fatalf("rivers = %+v", envelope.Data.Rivers)
	}
}

func TestRiverEndpointsSayWhenNothingIsCollectedYet(t *testing.T) {
	server := newServer(t, Dependencies{RiverRisk: &fakeRiverRisk{empty: true}})
	for _, path := range []string{"/v1/rivers/risk", "/v1/floods/basins"} {
		resp, body := get(t, server, path)
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("%s status %d", path, resp.StatusCode)
		}
		var envelope Envelope
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error == nil || envelope.Error.Code != "no_data" {
			t.Fatalf("%s: %s", path, body)
		}
	}
}

func TestRiverRiskFailureIsAnInternalError(t *testing.T) {
	server := newServer(t, Dependencies{RiverRisk: &fakeRiverRisk{err: errors.New("database is down")}})
	resp, body := get(t, server, "/v1/rivers/risk")
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var envelope Envelope
	_ = json.Unmarshal(body, &envelope)
	if envelope.Error == nil || envelope.Error.Code != "internal" {
		t.Fatalf("body = %s", body)
	}
}

func TestRiverDetailAndUnknownRiver(t *testing.T) {
	server := newServer(t, Dependencies{RiverRisk: &fakeRiverRisk{}})
	if resp, body := get(t, server, "/v1/rivers/kelani-ganga"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	resp, body := get(t, server, "/v1/rivers/nowhere")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var envelope Envelope
	_ = json.Unmarshal(body, &envelope)
	if envelope.Error == nil || envelope.Error.Code != "river_not_found" {
		t.Fatalf("body = %s", body)
	}
}

func TestExistingRiverRoutesStillResolve(t *testing.T) {
	server := newServer(t, Dependencies{Rivers: &fakeRivers{}, RiverRisk: &fakeRiverRisk{}})
	if resp, body := get(t, server, "/v1/rivers/levels"); resp.StatusCode != http.StatusOK {
		t.Fatalf("levels: status %d: %s", resp.StatusCode, body)
	}
	if resp, body := get(t, server, "/v1/rivers/stations/hanwella/readings"); resp.StatusCode != http.StatusOK {
		t.Fatalf("readings: status %d: %s", resp.StatusCode, body)
	}
}

func TestFloodBasinsAndUnknownBasin(t *testing.T) {
	server := newServer(t, Dependencies{RiverRisk: &fakeRiverRisk{}})
	resp, body := get(t, server, "/v1/floods/basins")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}
	if resp, _ := get(t, server, "/v1/floods/basins/kelani-ganga"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	resp, body = get(t, server, "/v1/floods/basins/nowhere")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var envelope Envelope
	_ = json.Unmarshal(body, &envelope)
	if envelope.Error == nil || envelope.Error.Code != "basin_not_found" {
		t.Fatalf("body = %s", body)
	}
}

func TestPastFloodsSayWhatTheyAreNot(t *testing.T) {
	server := newServer(t, Dependencies{RiverRisk: &fakeRiverRisk{}})
	resp, body := get(t, server, "/v1/floods/past")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var envelope struct {
		Data struct {
			Note string `json:"note"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Note != riverrisk.PastFloodNote {
		t.Fatalf("note = %q", envelope.Data.Note)
	}
}

func TestRiverTiles(t *testing.T) {
	server := newServer(t, Dependencies{RiverRisk: &fakeRiverRisk{}})
	for _, base := range []string{"/v1/tiles/rivers", "/v1/tiles/river-basins", "/v1/tiles/past-floods"} {
		resp, body := get(t, server, base+"/7/92/62.pbf")
		if resp.StatusCode != http.StatusOK || string(body) != "tile" {
			t.Fatalf("%s: status %d: %s", base, resp.StatusCode, body)
		}
		if resp.Header.Get("Content-Type") != tileContentType {
			t.Fatalf("%s: content type %q", base, resp.Header.Get("Content-Type"))
		}
		if resp, _ := get(t, server, base+"/3/1/1.pbf"); resp.StatusCode != http.StatusNoContent {
			t.Fatalf("%s: an empty tile is 204, got %d", base, resp.StatusCode)
		}
		if resp, _ := get(t, server, base+"/99/1/1.pbf"); resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s: an invalid tile is 400, got %d", base, resp.StatusCode)
		}
	}
}

func TestRiverEndpointsWithoutTheService(t *testing.T) {
	server := newServer(t, Dependencies{})
	for _, path := range []string{"/v1/rivers/risk", "/v1/rivers/x", "/v1/floods/basins", "/v1/floods/past", "/v1/tiles/rivers/7/92/62.pbf"} {
		if resp, _ := get(t, server, path); resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("%s: status %d", path, resp.StatusCode)
		}
	}
}
