package waterlevels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	// CanalCredit is the credit waterlevels.lk asks anyone reading this data to give.
	CanalCredit = "Data © CUrW / SLLDC (pub.curwsl.org)"
	// minimumCanals is the fewest a complete response has; it had 33 when written.
	minimumCanals = 15
)

// CanalPoint is one reading of a gauge's last 24 hours.
type CanalPoint struct {
	At     time.Time `json:"t"`
	LevelM float64   `json:"level_m"`
}

// CanalGauge is one gauge. The feed gives no coordinates: its position is not known.
type CanalGauge struct {
	ID, Name, Group string
	// State is the source's word: normal, rising, higher or offline.
	State      string
	LevelM     *float64
	ObservedAt *time.Time
	Change3hM  *float64
	Last24h    []CanalPoint
}

// TideEvent is a predicted high or low tide.
type TideEvent struct {
	Kind string    `json:"kind"`
	At   time.Time `json:"at"`
}

// CanalSnapshot is one decoded /api/canals response.
type CanalSnapshot struct {
	GeneratedAt          time.Time
	Credit, Licence, URL string
	SeaLevelM            *float64
	SeaTrend             *string
	SeaObservedAt        *time.Time
	TideNote             string
	Tide                 []TideEvent
	Gauges               []CanalGauge
	// Skipped counts gauges with no name.
	Skipped int
}

var canalSlug = regexp.MustCompile(`[^a-z0-9]+`)

// DecodeCanals turns an /api/canals response into gauges. It does no I/O.
func DecodeCanals(payload []byte) (CanalSnapshot, error) {
	var body struct {
		Credit      string `json:"credit"`
		Source      string `json:"source"`
		Licence     string `json:"licence"`
		GeneratedAt string `json:"generatedAt"`
		Sea         *struct {
			LevelM     *float64 `json:"level_m"`
			Trend      *string  `json:"trend"`
			ObservedAt *string  `json:"observedAt"`
		} `json:"sea"`
		Gauges []struct {
			Name       string   `json:"name"`
			Group      string   `json:"group"`
			State      string   `json:"state"`
			LevelM     *float64 `json:"level_m"`
			ObservedAt *string  `json:"observedAt"`
			Change3hM  *float64 `json:"change3h_m"`
			Last24h    []struct {
				At     string   `json:"t"`
				LevelM *float64 `json:"level_m"`
			} `json:"last24h"`
		} `json:"gauges"`
		PredictedTide *struct {
			Note    string `json:"note"`
			Next48h []struct {
				Kind string `json:"kind"`
				At   string `json:"at"`
			} `json:"next48h"`
		} `json:"predictedTide"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return CanalSnapshot{}, fmt.Errorf("decode canals: %w", err)
	}
	generated, err := time.Parse(time.RFC3339, body.GeneratedAt)
	if err != nil {
		return CanalSnapshot{}, errors.New("canals response has no generatedAt")
	}
	snapshot := CanalSnapshot{GeneratedAt: generated.UTC(), Credit: strings.TrimSpace(body.Credit), URL: strings.TrimSpace(body.Source), Licence: strings.TrimSpace(body.Licence)}
	if body.Sea != nil {
		snapshot.SeaLevelM, snapshot.SeaTrend, snapshot.SeaObservedAt = body.Sea.LevelM, body.Sea.Trend, canalTime(body.Sea.ObservedAt)
	}
	if body.PredictedTide != nil {
		snapshot.TideNote = strings.TrimSpace(body.PredictedTide.Note)
		for _, event := range body.PredictedTide.Next48h {
			if at := canalTime(&event.At); at != nil {
				snapshot.Tide = append(snapshot.Tide, TideEvent{Kind: event.Kind, At: *at})
			}
		}
	}
	seen := make(map[string]bool, len(body.Gauges))
	for _, wire := range body.Gauges {
		name := strings.TrimSpace(wire.Name)
		id := strings.Trim(canalSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
		if id == "" || seen[id] {
			snapshot.Skipped++
			continue
		}
		seen[id] = true
		gauge := CanalGauge{ID: id, Name: name, Group: strings.TrimSpace(wire.Group), State: strings.ToLower(strings.TrimSpace(wire.State)),
			LevelM: wire.LevelM, ObservedAt: canalTime(wire.ObservedAt), Change3hM: wire.Change3hM, Last24h: []CanalPoint{}}
		if gauge.State == "" {
			gauge.State = "unknown"
		}
		for _, point := range wire.Last24h {
			if at := canalTime(&point.At); at != nil && point.LevelM != nil {
				gauge.Last24h = append(gauge.Last24h, CanalPoint{At: *at, LevelM: *point.LevelM})
			}
		}
		snapshot.Gauges = append(snapshot.Gauges, gauge)
	}
	return snapshot, nil
}

func canalTime(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}

// CanalCollector stores Colombo's canal gauges, the sea level at the Kelani mouth
// and the predicted tide. The response replaces what is stored.
type CanalCollector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// Endpoint and Minimum override the defaults, for tests.
	Endpoint string
	Minimum  int
}

// Name implements source.Collector.
func (collector *CanalCollector) Name() string { return CanalsSource }

// Interval implements source.Collector. The gauges report every 15 minutes and the
// site caches for some of that; half an hour matches the other gauges from this source.
func (collector *CanalCollector) Interval() time.Duration { return 30 * time.Minute }

// Collect fetches the canals and returns the number of gauges stored.
func (collector *CanalCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	endpoint := collector.Endpoint
	if endpoint == "" {
		endpoint = CanalsURL
	}
	payload, err := collector.Fetcher.Fetch(ctx, source.Request{URL: endpoint})
	if err != nil {
		return 0, fmt.Errorf("fetch canals: %w", err)
	}
	snapshot, err := DecodeCanals(payload)
	if err != nil {
		return 0, err
	}
	minimum := minimumCanals
	if collector.Minimum > 0 {
		minimum = collector.Minimum
	}
	if len(snapshot.Gauges) < minimum {
		return 0, fmt.Errorf("source returned %d canal gauges, want at least %d", len(snapshot.Gauges), minimum)
	}
	tide, err := json.Marshal(append([]TideEvent{}, snapshot.Tide...))
	if err != nil {
		return 0, err
	}
	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		if err := queries.DeleteCanalGauges(ctx); err != nil {
			return err
		}
		for _, gauge := range snapshot.Gauges {
			last, err := json.Marshal(gauge.Last24h)
			if err != nil {
				return err
			}
			if err := queries.InsertCanalGauge(ctx, store.InsertCanalGaugeParams{ID: gauge.ID, Name: gauge.Name, CanalGroup: gauge.Group, State: gauge.State,
				LevelM: gauge.LevelM, ObservedAt: gauge.ObservedAt, Change3hM: gauge.Change3hM, Last24h: last, Source: CanalsSource,
				SourceTime: snapshot.GeneratedAt, FetchedAt: now}); err != nil {
				return fmt.Errorf("store canal gauge %s: %w", gauge.ID, err)
			}
		}
		return queries.PutCanalContext(ctx, store.PutCanalContextParams{Credit: snapshot.Credit, Licence: snapshot.Licence, UpstreamUrl: snapshot.URL,
			SeaLevelM: snapshot.SeaLevelM, SeaTrend: snapshot.SeaTrend, SeaObservedAt: snapshot.SeaObservedAt, TideNote: snapshot.TideNote, Tide: tide,
			Source: CanalsSource, SourceTime: snapshot.GeneratedAt, FetchedAt: now})
	})
	if err != nil {
		return 0, err
	}
	return len(snapshot.Gauges), nil
}
