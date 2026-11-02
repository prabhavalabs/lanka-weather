package rivernet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/hydrorivers"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/irrigationgis"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// SourceName identifies this derived dataset in source_runs and source_versions.
const SourceName = "river-network"

const (
	// googleRetention keeps Google's ids no longer than its terms allow.
	googleRetention = 24 * time.Hour
	// searchDegrees is the search radius for segments near a gauge, a little
	// beyond the largest limit (5 km is about 0.045°).
	searchDegrees = 0.055
	buildTimeout  = "3min"
)

// Collector rebuilds the network when its inputs change. It fetches nothing:
// it derives from the tables other collectors fill, so it runs last and costs
// a few small queries when nothing changed.
type Collector struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

// Name implements source.Collector.
func (collector *Collector) Name() string { return SourceName }

// Interval implements source.Collector. New gauges are rare, and Google's are
// replaced hourly.
func (collector *Collector) Interval() time.Duration { return time.Hour }

// Prune implements source.Pruner. A rebuild normally keeps these current; this
// holds Google's ids to the terms even if rebuilds stop.
func (collector *Collector) Prune(ctx context.Context, now time.Time) error {
	return store.New(collector.Pool).DeleteGoogleGaugeMatchesBefore(ctx, now.Add(-googleRetention))
}

// Collect rebuilds the network if the rivers, official names or gauges have
// changed, and returns the number of reaches stored. With no rivers loaded yet
// it does nothing, and reports no error: the next run picks them up.
func (collector *Collector) Collect(ctx context.Context, now time.Time) (int, error) {
	queries := store.New(collector.Pool)
	count, err := queries.CountRiverSegments(ctx)
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}
	stations, err := queries.ListNetworkStations(ctx)
	if err != nil {
		return 0, fmt.Errorf("list stations: %w", err)
	}
	gauges, err := queries.ListNetworkFloodGauges(ctx, now.Add(-googleRetention))
	if err != nil {
		return 0, fmt.Errorf("list flood gauges: %w", err)
	}
	fingerprint, err := collector.fingerprint(ctx, queries, stations, gauges)
	if err != nil {
		return 0, err
	}
	stored, err := queries.GetSourceVersion(ctx, SourceName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("read network version: %w", err)
	}
	if err == nil && stored.Version == fingerprint {
		// Nothing moved; Google's rows just need their time renewed.
		return 0, collector.touchGoogle(ctx, queries, now)
	}
	return collector.build(ctx, queries, stations, gauges, fingerprint, now)
}

// fingerprint hashes everything a build reads, so a rebuild happens exactly
// when one could change the result. River_stations is hashed whole, whichever
// collector added the row.
func (collector *Collector) fingerprint(ctx context.Context, queries *store.Queries, stations []store.ListNetworkStationsRow, gauges []store.ListNetworkFloodGaugesRow) (string, error) {
	parts := []string{"format-1"}
	for _, name := range []string{hydrorivers.SourceName, irrigationgis.BasinsSource} {
		version, err := queries.GetSourceVersion(ctx, name)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("read %s version: %w", name, err)
		}
		parts = append(parts, name+"="+version.Version)
	}
	for _, station := range stations {
		parts = append(parts, fmt.Sprintf("s:%s:%s:%s:%.5f:%.5f", station.ID, station.Name, station.Basin, station.Lon, station.Lat))
	}
	for _, gauge := range gauges {
		parts = append(parts, fmt.Sprintf("g:%s:%.5f:%.5f:%v", gauge.GaugeID, gauge.Lon, gauge.Lat, gauge.QualityVerified))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:16]), nil
}

func (collector *Collector) touchGoogle(ctx context.Context, queries *store.Queries, now time.Time) error {
	return queries.TouchGoogleGaugeMatches(ctx, now)
}

type station = store.ListNetworkStationsRow

func (collector *Collector) build(ctx context.Context, queries *store.Queries, stations []station, gauges []store.ListNetworkFloodGaugesRow, version string, now time.Time) (int, error) {
	topology, err := queries.ListSegmentTopology(ctx)
	if err != nil {
		return 0, fmt.Errorf("read segments: %w", err)
	}
	segments := make([]Segment, len(topology))
	for index, row := range topology {
		segments[index] = Segment{ID: row.HyrivID, NextDown: row.NextDown, MainRiv: row.MainRiv, LengthKM: row.LengthKm, UplandKM2: row.UplandKm2}
	}
	sourceRows, err := queries.ListRiverSourceNames(ctx)
	if err != nil {
		return 0, fmt.Errorf("read river names: %w", err)
	}
	sourceNames := make(map[int64]string, len(sourceRows))
	for _, row := range sourceRows {
		sourceNames[row.RivID] = row.Name
	}
	voteRows, err := queries.SystemBasinVotes(ctx)
	if err != nil {
		return 0, fmt.Errorf("match rivers to basins: %w", err)
	}
	votes := make([]BasinVote, len(voteRows))
	for index, row := range voteRows {
		votes[index] = BasinVote{MainRiv: row.MainRiv, BasinFID: row.BasinFid, Basin: row.BasinName, CatchKM2: row.CatchKm2}
	}
	systemNames := ResolveSystemNames(votes)

	// Which system each name means, for a station's basin.
	systems := map[string]int64{}
	mainNames := map[int64]string{}
	for _, segment := range segments {
		if segment.NextDown != 0 {
			continue
		}
		for _, name := range []string{sourceNames[segment.ID], displayNames[systemNames[segment.ID].Name], systemNames[segment.ID].Name} {
			if name != "" {
				systems[Fold(name)] = segment.MainRiv
				mainNames[segment.MainRiv] = name
			}
		}
	}

	stems := MainStems(segments)
	officialRows, err := queries.ListOfficialStations(ctx)
	if err != nil {
		return 0, fmt.Errorf("read official stations: %w", err)
	}
	onMainRiver := map[string]bool{}
	for _, row := range officialRows {
		if row.Tributary != "" && CanonicalBasin(row.Tributary) == CanonicalBasin(row.Basin) {
			onMainRiver[Fold(row.Station)] = true
		}
	}

	var placed []Gauge
	var matches []store.InsertRiverGaugeMatchParams
	stationSegments := map[string]int64{}
	var unmatched []string
	for _, row := range stations {
		candidates, err := candidatesNear(ctx, queries, row.Lon, row.Lat)
		if err != nil {
			return 0, err
		}
		hint := StationHint{System: systems[CanonicalBasin(row.Basin)]}
		if onMainRiver[Fold(row.Name)] {
			hint.MainStem = stems[hint.System]
		}
		chosen, method, ok := ChooseStationSegment(candidates, hint)
		if !ok {
			unmatched = append(unmatched, "station "+row.ID+" ("+row.Source+")")
			continue
		}
		placed = append(placed, Gauge{Kind: "station", ID: row.ID, Label: row.Name, SegmentID: chosen.SegmentID})
		stationSegments[Fold(row.Name)] = chosen.SegmentID
		matches = append(matches, store.InsertRiverGaugeMatchParams{Kind: "station", GaugeID: row.ID, HyrivID: chosen.SegmentID,
			DistanceM: chosen.DistanceM, Method: method, MatchedAt: now})
		if method == "position" && collector.Logger != nil {
			collector.Logger.Info("station placed by position alone", "station", row.ID, "basin", row.Basin, "source", row.Source)
		}
	}
	for _, row := range gauges {
		candidates, err := candidatesNear(ctx, queries, row.Lon, row.Lat)
		if err != nil {
			return 0, err
		}
		chosen, method, ok := ChooseGoogleSegment(candidates, HybasOfGauge(row.GaugeID))
		if !ok {
			unmatched = append(unmatched, "google "+row.GaugeID)
			continue
		}
		// Unverified gauges tint basins (they are read there); they do not
		// govern reaches, whose risk reads verified forecasts only.
		if row.QualityVerified {
			placed = append(placed, Gauge{Kind: "google", ID: row.GaugeID, Label: "Flood Hub point", SegmentID: chosen.SegmentID})
		}
		matches = append(matches, store.InsertRiverGaugeMatchParams{Kind: "google", GaugeID: row.GaugeID, HyrivID: chosen.SegmentID,
			DistanceM: chosen.DistanceM, Method: method, MatchedAt: now})
	}
	if len(unmatched) > 0 && collector.Logger != nil {
		collector.Logger.Warn("gauges not on the river network", "gauges", unmatched)
	}

	tributaries, err := collector.officialTributaries(ctx, queries, officialRows, segments, stationSegments, mainNames)
	if err != nil {
		return 0, err
	}
	network := Build(Inputs{Segments: segments, SystemNames: systemNames, SourceNames: sourceNames, Tributaries: tributaries, Gauges: placed})
	if err := collector.save(ctx, network, matches, version, now); err != nil {
		return 0, err
	}
	if collector.Logger != nil {
		collector.Logger.Info("river network built", "rivers", len(network.Rivers), "reaches", len(network.Reaches),
			"gauges_placed", len(placed), "gauges_unmatched", len(unmatched))
	}
	return len(network.Reaches), nil
}

func candidatesNear(ctx context.Context, queries *store.Queries, lon, lat float64) ([]Candidate, error) {
	rows, err := queries.SegmentsNearPoint(ctx, store.SegmentsNearPointParams{Lon: lon, Lat: lat, Degrees: searchDegrees})
	if err != nil {
		return nil, fmt.Errorf("segments near a gauge: %w", err)
	}
	candidates := make([]Candidate, len(rows))
	for index, row := range rows {
		candidates[index] = Candidate{SegmentID: row.HyrivID, MainRiv: row.MainRiv, HybasL12: row.HybasL12, UplandKM2: row.UplandKm2, DistanceM: row.DistanceM}
	}
	return candidates, nil
}

// officialTributaries reads tributary names from the department: first the
// tributary its station list gives each gauge, then the named stream polygons.
func (collector *Collector) officialTributaries(ctx context.Context, queries *store.Queries, rows []store.ListOfficialStationsRow, segments []Segment, stationSegments map[string]int64, mainNames map[int64]string) ([]Tributary, error) {
	var tributaries []Tributary
	seen := map[string]bool{}
	for _, row := range rows {
		segment, ok := stationSegments[Fold(row.Station)]
		if !ok || row.Tributary == "" || CanonicalBasin(row.Tributary) == CanonicalBasin(row.Basin) {
			continue
		}
		if !seen[Fold(row.Tributary)] {
			seen[Fold(row.Tributary)] = true
			tributaries = append(tributaries, Tributary{SourceID: SourceAbove(segments, segment), Name: row.Tributary})
		}
	}
	voteRows, err := queries.StreamNameVotes(ctx, minNamedUplandKM2/4)
	if err != nil {
		return nil, fmt.Errorf("match streams to names: %w", err)
	}
	votes := make([]StreamVote, len(voteRows))
	for index, row := range voteRows {
		votes[index] = StreamVote{SegmentID: row.HyrivID, Name: row.Name}
	}
	for _, tributary := range TributariesFromVotes(segments, votes, mainNames) {
		if !seen[Fold(tributary.Name)] {
			seen[Fold(tributary.Name)] = true
			tributaries = append(tributaries, tributary)
		}
	}
	return tributaries, nil
}

func (collector *Collector) save(ctx context.Context, network Network, matches []store.InsertRiverGaugeMatchParams, version string, now time.Time) error {
	ids := make([]int64, 0, len(network.SegmentRiver))
	for id := range network.SegmentRiver {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	rivers, reaches := make([]string, len(ids)), make([]int64, len(ids))
	for index, id := range ids {
		rivers[index] = network.SegmentRiver[id]
		reaches[index] = network.SegmentReach[id]
	}
	riverOfSegment := network.SegmentRiver
	return pgx.BeginFunc(ctx, collector.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '"+buildTimeout+"'"); err != nil {
			return err
		}
		queries := store.New(tx)
		if err := queries.ClearSegmentAssignments(ctx); err != nil {
			return err
		}
		if err := queries.DeleteRivers(ctx); err != nil {
			return err
		}
		if err := queries.DeleteRiverGaugeMatches(ctx); err != nil {
			return err
		}
		for _, river := range network.Rivers {
			row := store.InsertRiverParams{
				ID: river.ID, Name: river.Name, Aliases: river.Aliases, Named: river.Named, MainRiv: river.MainRiv,
				MouthID: river.MouthID, LengthKm: river.LengthKM, UplandKm2: river.UplandKM2, BuiltAt: now,
			}
			if row.Aliases == nil {
				row.Aliases = []string{}
			}
			if river.NameSI != "" {
				row.NameSi = &river.NameSI
			}
			if river.ParentID != "" {
				row.ParentID = &river.ParentID
			}
			if river.BasinFID != 0 {
				row.BasinFid = &river.BasinFID
			}
			if err := queries.InsertRiver(ctx, row); err != nil {
				return fmt.Errorf("store river %s: %w", river.ID, err)
			}
		}
		for _, reach := range network.Reaches {
			if err := queries.InsertRiverReach(ctx, store.InsertRiverReachParams{
				Fid: reach.FID, RiverID: reach.RiverID, Name: reach.Name, Gauged: reach.Gauged(),
				LengthKm: reach.LengthKM, UplandKm2: reach.UplandKM2, Position: int32(reach.Position),
			}); err != nil {
				return fmt.Errorf("store reach %d: %w", reach.FID, err)
			}
			insert := func(ref GaugeRef, role string) error {
				return queries.InsertRiverReachGauge(ctx, store.InsertRiverReachGaugeParams{ReachFid: reach.FID, Kind: ref.Kind, GaugeID: ref.ID, Role: role})
			}
			for _, ref := range reach.Upstream {
				if err := insert(ref, "upstream"); err != nil {
					return err
				}
			}
			if reach.Downstream != nil {
				if err := insert(*reach.Downstream, "downstream"); err != nil {
					return err
				}
			}
			if reach.Next != nil {
				if err := insert(*reach.Next, "next"); err != nil {
					return err
				}
			}
		}
		for _, match := range matches {
			if river, ok := riverOfSegment[match.HyrivID]; ok {
				match.RiverID = &river
			}
			if err := queries.InsertRiverGaugeMatch(ctx, match); err != nil {
				return fmt.Errorf("store match %s: %w", match.GaugeID, err)
			}
		}
		if err := queries.AssignSegments(ctx, store.AssignSegmentsParams{Ids: ids, RiverIds: rivers, ReachFids: reaches}); err != nil {
			return fmt.Errorf("assign segments: %w", err)
		}
		if _, err := queries.BuildReachGeometry(ctx); err != nil {
			return fmt.Errorf("merge reach lines: %w", err)
		}
		if _, err := queries.BuildReachParts(ctx); err != nil {
			return fmt.Errorf("build drawn lines: %w", err)
		}
		if _, err := queries.BuildRiverExtents(ctx); err != nil {
			return fmt.Errorf("build river extents: %w", err)
		}
		if _, err := queries.NameUnnamedRivers(ctx); err != nil {
			return fmt.Errorf("name unnamed rivers: %w", err)
		}
		return queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: SourceName, Version: version, SourceTime: now, FetchedAt: now, Records: int32(len(network.Reaches)),
		})
	})
}
