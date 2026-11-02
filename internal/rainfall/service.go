// Package rainfall serves the IMERG satellite rain estimate from the tiles
// the imerg collector stores. It never calls NASA for a request.
package rainfall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/imerg"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// ErrTileNotFound means no tile is stored for that time and place.
var ErrTileNotFound = errors.New("no rain tile for that time and place")

// Service reads stored tiles.
type Service struct {
	Pool *pgxpool.Pool
	// StaleAfter is the image age after which the layer is marked stale.
	StaleAfter time.Duration
	Now        func() time.Time
}

// Steps is GET /v1/rain/steps: what the layer needs to draw and explain the
// images. Whether a step is usable is decided by the collector, which stores a
// step whole or not at all.
type Steps struct {
	Source      string `json:"source"`
	Attribution string `json:"attribution"`
	// TilePath is the tile URL template, with {time}, {z}, {x} and {y}.
	TilePath string `json:"tilePath"`
	MinZoom  int    `json:"minZoom"`
	MaxZoom  int    `json:"maxZoom"`
	TileSize int    `json:"tileSize"`
	// Bounds is west, south, east, north.
	Bounds [4]float64 `json:"bounds"`
	// Steps are oldest first, so a slider runs towards the present.
	Steps []Step `json:"steps"`
	// NewestValidAt is the start of the newest image's half hour.
	NewestValidAt *time.Time `json:"newestValidAt"`
	FetchedAt     *time.Time `json:"fetchedAt"`
	Stale         bool       `json:"stale"`
	Estimate      bool       `json:"estimate"`
	Legend        Legend     `json:"legend"`
	Note          string     `json:"note"`
}

// Step is one half hour's images.
type Step struct {
	// Time goes in the tile URL.
	Time string `json:"time"`
	// ValidAt and EndsAt bound the half hour the image covers.
	ValidAt   time.Time `json:"validAt"`
	EndsAt    time.Time `json:"endsAt"`
	FetchedAt time.Time `json:"fetchedAt"`
	// EmptyTiles of Tiles show no rain anywhere.
	Tiles      int `json:"tiles"`
	EmptyTiles int `json:"emptyTiles"`
}

// Legend is the colour scale NASA renders the images with.
type Legend struct {
	Unit  string             `json:"unit"`
	Stops []imerg.LegendStop `json:"stops"`
	// Source is the colour map the stops come from.
	Source string `json:"source"`
}

const note = "Satellite estimate from NASA's IMERG, not a measurement. Images appear hours after the time they show."

// Steps lists the stored half hours.
func (service *Service) Steps(ctx context.Context) (*Steps, error) {
	rows, err := store.New(service.Pool).RainSteps(ctx)
	if err != nil {
		return nil, fmt.Errorf("read rain steps: %w", err)
	}
	inputs := make([]Step, len(rows))
	for index, row := range rows {
		inputs[index] = Step{Time: imerg.TimeID(row.ValidAt), ValidAt: row.ValidAt.UTC(), EndsAt: row.ValidAt.UTC().Add(imerg.Step),
			FetchedAt: row.FetchedAt.UTC(), Tiles: int(row.Tiles), EmptyTiles: int(row.EmptyTiles)}
	}
	return BuildSteps(inputs, service.now(), service.StaleAfter), nil
}

// BuildSteps assembles the answer from stored steps in any order. A layer with
// no images is stale: there is nothing current to show.
func BuildSteps(steps []Step, now time.Time, staleAfter time.Duration) *Steps {
	sort.Slice(steps, func(i, j int) bool { return steps[i].ValidAt.Before(steps[j].ValidAt) })
	out := &Steps{
		Source: imerg.SourceName, Attribution: imerg.Attribution,
		TilePath: "/v1/rain/tiles/{time}/{z}/{x}/{y}.png", MinZoom: imerg.Zoom, MaxZoom: imerg.Zoom, TileSize: 256,
		Bounds: imerg.Bounds, Steps: steps, Stale: true, Estimate: true, Note: note,
		Legend: Legend{Unit: "mm/h", Stops: imerg.LegendStops, Source: imerg.ColormapURL},
	}
	if out.Steps == nil {
		out.Steps = []Step{}
	}
	if len(steps) > 0 {
		newest := steps[len(steps)-1]
		validAt, fetchedAt := newest.ValidAt, newest.FetchedAt
		out.NewestValidAt, out.FetchedAt = &validAt, &fetchedAt
		out.Stale = now.Sub(newest.EndsAt) > staleAfter
	}
	return out
}

// Tile returns one tile's PNG. A stored tile with no rain gives a transparent
// image, so the map treats it like any other.
func (service *Service) Tile(ctx context.Context, timeID string, z, x, y int) ([]byte, error) {
	validAt, err := imerg.ParseTimeID(timeID)
	if err != nil {
		return nil, ErrTileNotFound
	}
	if !imerg.Covers(imerg.Tile{Z: z, X: x, Y: y}) {
		return nil, ErrTileNotFound
	}
	row, err := store.New(service.Pool).RainTile(ctx, store.RainTileParams{ValidAt: validAt, Z: int32(z), X: int32(x), Y: int32(y)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTileNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read rain tile: %w", err)
	}
	if row.Png == nil {
		return EmptyTile(), nil
	}
	return row.Png, nil
}

// EmptyTile is a one-pixel transparent PNG.
var EmptyTile = sync.OnceValue(func() []byte {
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		panic(err)
	}
	return out.Bytes()
})

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
