package regions

import (
	"testing"

	"github.com/prabhavalabs/lanka-weather/internal/layers"
)

func TestCatalogueMatchesTileZooms(t *testing.T) {
	layer, ok := layers.ByID("admin-regions")
	if !ok || layer.MaxZoom != TileMaxZoom || layer.MinZoom != TileMinZoom[LevelProvince] {
		t.Fatalf("catalogue zooms %d to %d, tiles %d to %d", layer.MinZoom, layer.MaxZoom, TileMinZoom[LevelProvince], TileMaxZoom)
	}
}
