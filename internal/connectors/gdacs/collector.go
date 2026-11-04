package gdacs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// DefaultBaseURL is the GDACS API.
const DefaultBaseURL = "https://www.gdacs.org/gdacsapi/api"

// Collector stores current North Indian Ocean cyclones.
type Collector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// BaseURL overrides DefaultBaseURL, for tests.
	BaseURL string
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector. Advisories come about every six hours.
func (collector *Collector) Interval() time.Duration { return time.Hour }

// Collect reads the list and each current event's geometry. Having nothing to
// store is an answer too: the fetch time recorded here is what lets the map
// say "no active cyclone" with an age. A geometry that fails to load keeps the
// event, with its last geometry, and the run reports the error.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	base := collector.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	payload, err := collector.Fetcher.Fetch(ctx, source.Request{
		URL:   base + "/events/geteventlist/SEARCH",
		Query: url.Values{"eventlist": {"TC"}, "alertlevel": {"green;orange;red"}},
	})
	if err != nil {
		return 0, fmt.Errorf("fetch cyclone list: %w", err)
	}
	events, _, err := DecodeEvents(payload)
	if err != nil {
		return 0, err
	}
	relevant := Relevant(events, now)

	var failures []error
	features := make(map[int64][]byte, len(relevant))
	for _, event := range relevant {
		raw, err := collector.geometry(ctx, base, event)
		if err != nil {
			failures = append(failures, fmt.Errorf("geometry of %s: %w", event.Name, err))
			raw = []byte("[]")
		}
		features[event.ID] = raw
	}

	err = pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		queries := store.New(tx)
		keep := make([]int64, 0, len(relevant))
		for _, event := range relevant {
			keep = append(keep, event.ID)
			if err := queries.UpsertCyclone(ctx, store.UpsertCycloneParams{
				EventID: event.ID, EpisodeID: int32(event.Episode), Name: event.Name, AlertLevel: event.AlertLevel,
				WindKmh: event.WindKmh, SeverityText: event.SeverityText, Lon: event.Lon, Lat: event.Lat,
				Country: event.Country, StartedAt: event.StartedAt, LatestAt: event.LatestAt,
				AdvisorySource: event.AdvisorySource, ReportUrl: event.ReportURL, IsCurrent: true,
				Features: features[event.ID], Source: SourceName, SourceTime: event.ModifiedAt, FetchedAt: now,
			}); err != nil {
				return fmt.Errorf("store %s: %w", event.Name, err)
			}
		}
		if err := queries.ClearCurrentCyclones(ctx, keep); err != nil {
			return fmt.Errorf("clear ended cyclones: %w", err)
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: SourceName, Version: now.UTC().Format(time.RFC3339), SourceTime: now,
			FetchedAt: now, Records: int32(len(relevant)),
		})
	})
	if err != nil {
		return 0, err
	}
	return len(relevant), errors.Join(failures...)
}

func (collector *Collector) geometry(ctx context.Context, base string, event Event) ([]byte, error) {
	payload, err := collector.Fetcher.Fetch(ctx, source.Request{
		URL: base + "/polygons/getgeometry",
		Query: url.Values{
			"eventtype": {"TC"},
			"eventid":   {strconv.FormatInt(event.ID, 10)},
			"episodeid": {strconv.Itoa(event.Episode)},
		},
	})
	if err != nil {
		return nil, err
	}
	features, err := DecodeGeometry(payload, event.LatestAt)
	if err != nil {
		return nil, err
	}
	return json.Marshal(features)
}
