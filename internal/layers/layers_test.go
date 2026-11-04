package layers

import "testing"

func TestCatalogueIsValid(t *testing.T) {
	if err := Validate(All()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsDuplicateIDs(t *testing.T) {
	layer := All()[0]
	if err := Validate([]Layer{layer, layer}); err == nil {
		t.Fatal("expected an error for duplicate ids")
	}
}

func TestValidateRejectsStaleLimitBelowRefresh(t *testing.T) {
	layer := All()[0]
	layer.StaleAfterSeconds = layer.RefreshSeconds - 1
	if err := Validate([]Layer{layer}); err == nil {
		t.Fatal("expected an error when the stale limit is below the refresh interval")
	}
}

func TestAllReturnsACopy(t *testing.T) {
	first := All()
	first[0].Sources[0] = "changed"
	if All()[0].Sources[0] == "changed" {
		t.Fatal("All must not expose the catalogue's backing slices")
	}
}

func TestByID(t *testing.T) {
	if _, ok := ByID("river-levels"); !ok {
		t.Fatal("river-levels should exist")
	}
	if _, ok := ByID("missing"); ok {
		t.Fatal("missing should not exist")
	}
}

func TestAdminRegionsLayerIsAvailableAsTiles(t *testing.T) {
	layer, ok := ByID("admin-regions")
	if !ok || layer.Status != StatusAvailable || layer.TilePath != "/v1/tiles/regions/{z}/{x}/{y}.pbf" || layer.MaxZoom != 14 {
		t.Fatalf("admin-regions = %+v", layer)
	}
	layer.TilePath = ""
	if err := Validate([]Layer{layer}); err == nil {
		t.Fatal("expected an error for an available vector layer without a tile path")
	}
}

func TestWeatherLayersAreAvailable(t *testing.T) {
	cases := map[string]string{
		"wind":             "/v1/weather/wind",
		"flood-forecast":   "/v1/floods/forecast",
		"temperature":      "/v1/weather/current",
		"rainfall":         "/v1/rain/steps",
		"weather-stations": "/v1/stations",
		"air-quality":      "/v1/air/current",
		"sea-conditions":   "/v1/sea/current",
	}
	for id, path := range cases {
		layer, ok := ByID(id)
		if !ok || layer.Status != StatusAvailable || layer.DataPath != path {
			t.Errorf("%s = %+v", id, layer)
		}
	}
	temperature, _ := ByID("temperature")
	if temperature.Kind != KindGrid || temperature.RefreshSeconds != 3600 {
		t.Fatalf("temperature = %+v", temperature)
	}
}

func TestValidateRequiresDataPathForAPILayers(t *testing.T) {
	layer, _ := ByID("wind")
	layer.DataPath = ""
	if err := Validate([]Layer{layer}); err == nil {
		t.Fatal("an available wind layer without a data path must be rejected")
	}
}
