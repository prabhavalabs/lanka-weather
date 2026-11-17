package floodrisk

import (
	"context"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/store"
	"github.com/prabhavalabs/lanka-weather/internal/warnings"
)

// Locator finds districts from the regions table, remembering points and areas:
// gauges do not move, and Google's areas are immutable under their IDs. The
// alert engine and the estimate share it so they place things alike.
type Locator struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger

	mu     sync.Mutex
	points map[string]string
	areas  map[string][]string
	basins *basinDistricts
}

func (locator *Locator) logger() *slog.Logger {
	if locator.Logger != nil {
		return locator.Logger
	}
	return slog.Default()
}

// DistrictOf is the district containing a point, or the nearest within about
// 5 km; "" when none is. A failed lookup is not remembered.
func (locator *Locator) DistrictOf(ctx context.Context, lon, lat float64) string {
	key := strconv.FormatFloat(lon, 'f', 5, 64) + "," + strconv.FormatFloat(lat, 'f', 5, 64)
	locator.mu.Lock()
	if district, ok := locator.points[key]; ok {
		locator.mu.Unlock()
		return district
	}
	locator.mu.Unlock()
	found, err := store.New(locator.Pool).DistrictAt(ctx, store.DistrictAtParams{Lon: lon, Lat: lat})
	if err != nil {
		locator.logger().Warn("district of a point failed", "error", err)
		return ""
	}
	district := ""
	if len(found) > 0 {
		district = found[0]
	}
	locator.mu.Lock()
	if locator.points == nil {
		locator.points = map[string]string{}
	}
	locator.points[key] = district
	locator.mu.Unlock()
	return district
}

// AreaDistricts are the districts a stored flood area overlaps.
func (locator *Locator) AreaDistricts(ctx context.Context, id string) []string {
	locator.mu.Lock()
	if districts, ok := locator.areas[id]; ok {
		locator.mu.Unlock()
		return districts
	}
	locator.mu.Unlock()
	districts, err := store.New(locator.Pool).FloodAreaDistricts(ctx, id)
	if err != nil {
		locator.logger().Warn("districts of a flood area failed", "error", err)
		return nil
	}
	locator.mu.Lock()
	if locator.areas == nil {
		locator.areas = map[string][]string{}
	}
	locator.areas[id] = districts
	locator.mu.Unlock()
	return districts
}

// Named matches districts by their English names, read once per call.
func (locator *Locator) Named(ctx context.Context) func(name string) string {
	byName := map[string]string{}
	if rows, err := store.New(locator.Pool).ListDistrictNames(ctx); err == nil {
		for _, row := range rows {
			byName[NormaliseName(row.Name)] = row.ID
		}
	} else {
		locator.logger().Warn("read district names failed", "error", err)
	}
	return func(name string) string { return byName[NormaliseName(name)] }
}

// NormaliseName compares district names whatever their spacing and case.
func NormaliseName(name string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// basinShareMin is the least share of a basin a district must hold to be
// named for a warning on that basin.
const basinShareMin = 0.05

// basinDistricts are each basin's districts, by basin key, read once.
type basinDistricts struct {
	once   sync.Once
	keys   []string
	byKey  map[string][]string
	failed bool
}

// BasinDistricts are the districts of the basins a flood warning's title
// names, in code order: "Flood Warning for Gin River Basin" is Galle and
// Matara. Nil when the title names no basin we know.
func (locator *Locator) BasinDistricts(ctx context.Context, title string) []string {
	locator.mu.Lock()
	if locator.basins == nil || locator.basins.failed {
		locator.basins = &basinDistricts{}
	}
	basins := locator.basins
	locator.mu.Unlock()
	basins.once.Do(func() {
		rows, err := store.New(locator.Pool).BasinDistrictShares(ctx)
		if err != nil {
			locator.logger().Warn("districts of basins failed", "error", err)
			basins.failed = true
			return
		}
		basins.byKey = map[string][]string{}
		for _, row := range rows {
			key := warnings.BasinKey(row.Basin)
			if key == "" || row.Share < basinShareMin {
				continue
			}
			if _, ok := basins.byKey[key]; !ok {
				basins.keys = append(basins.keys, key)
			}
			if !slices.Contains(basins.byKey[key], row.District) {
				basins.byKey[key] = append(basins.byKey[key], row.District)
			}
		}
	})
	if basins.failed {
		return nil
	}
	found := []string{}
	for _, key := range warnings.NamedBasins(title, basins.keys) {
		for _, district := range basins.byKey[key] {
			if !slices.Contains(found, district) {
				found = append(found, district)
			}
		}
	}
	if len(found) == 0 {
		return nil
	}
	slices.Sort(found)
	return found
}
