package warnings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/connectors/dmcbulletins"
	"github.com/prabhavalabs/lanka-weather/internal/connectors/landslides"
	"github.com/prabhavalabs/lanka-weather/internal/regions"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

const (
	defaultLimit = 60
	maxLimit     = 200
	// listWindow bounds the bulletin list and the search for ones in force.
	listWindow = 45 * 24 * time.Hour
	// inForceWindow is the longest period a bulletin has been seen to state.
	inForceWindow = 10 * 24 * time.Hour
	// districtWindow is how recent a bulletin naming a district must be to
	// be listed for a place in it.
	districtWindow = 48 * time.Hour
	historyRows    = 30
	bulletinNote   = "Bulletins reach us through nuuuwan/lk_dmc, usually within hours of being issued. The PDF on dmc.gov.lk is the official version."
)

// ErrDivisionNotFound is returned for a code that is not a known DS division.
var ErrDivisionNotFound = errors.New("division not found")

// Service reads official warnings from the database.
type Service struct {
	Pool *pgxpool.Pool
	// LandslideStaleAfter and BulletinStaleAfter are the catalogue's limits.
	LandslideStaleAfter time.Duration
	BulletinStaleAfter  time.Duration
	Now                 func() time.Time
}

// LandslideLayer is the landslide-warnings layer: the named divisions'
// simplified boundaries as GeoJSON, with their label points.
type LandslideLayer struct {
	Type     string            `json:"type"`
	Features []DivisionFeature `json:"features"`
	Meta     LandslideMeta     `json:"meta"`
}

// DivisionFeature is a division named in the report shown.
type DivisionFeature struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties DivisionAt      `json:"properties"`
}

// DivisionAt is a division's level in the report shown.
type DivisionAt struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	NameSi     *string `json:"nameSi"`
	DistrictID string  `json:"districtId"`
	District   string  `json:"district"`
	// LabelLon and LabelLat are a point inside the division.
	LabelLon float64 `json:"labelLon"`
	LabelLat float64 `json:"labelLat"`
	LevelWords
}

// LandslideMeta says what the layer's data is and what can be concluded from it.
type LandslideMeta struct {
	Source      string `json:"source"`
	Attribution string `json:"attribution"`
	// Status is waiting, active, lapsed, pending, none or unknown.
	Status string `json:"status"`
	// Note is the sentence to show about the status.
	Note string `json:"note"`
	// Stale says the feed has not been checked recently enough to trust.
	Stale bool `json:"stale"`
	// ReportDate is the latest report's local date, YYYY-MM-DD.
	ReportDate     *string    `json:"reportDate"`
	IssuedAt       *time.Time `json:"issuedAt"`
	ValidUntil     *time.Time `json:"validUntil"`
	ValidityStated bool       `json:"validityStated"`
	MessageID      *string    `json:"messageId"`
	PDFURL         *string    `json:"pdfUrl"`
	SourcePageURL  *string    `json:"sourcePageUrl"`
	// Counts are the divisions at each level of the latest report, shown or not.
	Counts map[string]int `json:"counts"`
	// Unresolved lists codes in the report that match no region.
	Unresolved  []string   `json:"unresolved"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	UpstreamAt  *time.Time `json:"upstreamAt"`
	GeneratedAt time.Time  `json:"generatedAt"`
}

// loaded is the latest report with everything needed to speak about it.
type loaded struct {
	meta      LandslideMeta
	divisions []DivisionFeature
	report    *store.LandslideReport
	show      bool
}

func (service *Service) loadLandslides(ctx context.Context) (loaded, error) {
	now := service.now()
	queries := store.New(service.Pool)
	result := loaded{meta: LandslideMeta{
		Source: landslides.SourceName, Attribution: landslides.Attribution, Status: StatusWaiting,
		Note:   "Landslide warnings have not been collected yet.",
		Counts: map[string]int{"1": 0, "2": 0, "3": 0}, Unresolved: []string{}, GeneratedAt: now,
	}}
	if version, err := queries.GetSourceVersion(ctx, landslides.SourceName); err == nil {
		fetched := version.FetchedAt
		result.meta.FetchedAt = &fetched
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return result, fmt.Errorf("read landslide version: %w", err)
	}
	if version, err := queries.GetSourceVersion(ctx, dmcbulletins.SourceName); err == nil {
		upstream := version.SourceTime
		result.meta.UpstreamAt = &upstream
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return result, fmt.Errorf("read bulletin version: %w", err)
	}
	report, err := queries.LatestLandslideReport(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, fmt.Errorf("read latest report: %w", err)
	}
	result.report = &report
	date := report.ReportDate.Time

	// The report's bulletin gives its issue time, period and official PDF.
	recent, err := queries.LandslideBulletinsBetween(ctx, date.Add(-24*time.Hour))
	if err != nil {
		return result, fmt.Errorf("read landslide bulletins: %w", err)
	}
	facts := ReportFacts{Date: date, CheckedAt: result.meta.FetchedAt, UpstreamAt: result.meta.UpstreamAt}
	var match, newer *store.OfficialBulletin
	for index := range recent {
		bulletin := &recent[index]
		local := DateInSriLanka(bulletin.IssuedAt)
		switch {
		case local.Equal(date) && match == nil:
			match = bulletin
		case local.After(date) && newer == nil:
			newer = bulletin
		}
	}
	if match != nil {
		issued := match.IssuedAt
		facts.IssuedAt = &issued
		facts.ValidUntil = match.ValidUntil
		result.meta.MessageID = match.MessageID
	}
	shown := match
	if newer != nil {
		issued := newer.IssuedAt
		facts.NewerIssuedAt = &issued
		shown = newer
	}
	if shown != nil {
		result.meta.PDFURL, result.meta.SourcePageURL = &shown.PdfUrl, &shown.SourcePageUrl
	}

	assessment := Assess(facts, now, service.LandslideStaleAfter)
	dateText := date.Format(time.DateOnly)
	result.meta.Status, result.meta.Note, result.meta.Stale = assessment.Status, assessment.Note, assessment.Stale
	result.meta.ReportDate = &dateText
	issuedAt, validUntil := assessment.IssuedAt, assessment.ValidUntil
	result.meta.IssuedAt, result.meta.ValidUntil, result.meta.ValidityStated = &issuedAt, &validUntil, assessment.ValidityStated
	result.show = assessment.ShowLevels

	rows, err := queries.LandslideReportLevels(ctx, report.ReportDate)
	if err != nil {
		return result, fmt.Errorf("read report levels: %w", err)
	}
	for _, row := range rows {
		result.meta.Counts[fmt.Sprint(row.Level)]++
		if row.DsName == nil {
			result.meta.Unresolved = append(result.meta.Unresolved, row.DsID)
			continue
		}
		district := ""
		if row.DistrictName != nil {
			district = *row.DistrictName
		}
		result.divisions = append(result.divisions, DivisionFeature{
			Type: "Feature", ID: row.DsID, Geometry: json.RawMessage(row.Geojson),
			Properties: DivisionAt{ID: row.DsID, Name: *row.DsName, NameSi: row.DsNameSi, DistrictID: row.DistrictID,
				District: district, LabelLon: row.Lon, LabelLat: row.Lat, LevelWords: WordsFor(int(row.Level))},
		})
	}
	return result, nil
}

// Landslides returns the landslide-warnings layer. Divisions are listed only
// while the report's levels belong on the map.
func (service *Service) Landslides(ctx context.Context) (LandslideLayer, error) {
	state, err := service.loadLandslides(ctx)
	if err != nil {
		return LandslideLayer{}, err
	}
	layer := LandslideLayer{Type: "FeatureCollection", Features: []DivisionFeature{}, Meta: state.meta}
	if state.show {
		layer.Features = append(layer.Features, state.divisions...)
	}
	return layer, nil
}

// HistoryEntry is a division's level in a past report.
type HistoryEntry struct {
	Date string `json:"date"`
	LevelWords
}

// Division is one division's landslide warning detail.
type Division struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	NameSi     *string `json:"nameSi"`
	DistrictID string  `json:"districtId"`
	District   string  `json:"district"`
	Lon        float64 `json:"lon"`
	Lat        float64 `json:"lat"`
	// Level is the level in the report currently shown; null if the division
	// is not named in it or no report is in force.
	Level *LevelWords `json:"level"`
	// Report says what the latest report is and what can be concluded.
	Report LandslideMeta `json:"report"`
	// History is the reports that named the division, newest first.
	History []HistoryEntry `json:"history"`
}

// Division returns a division's level and recent history.
func (service *Service) Division(ctx context.Context, id string) (*Division, error) {
	canonical, level, ok := regions.ParseID(id)
	if !ok || level != regions.LevelDS {
		return nil, ErrDivisionNotFound
	}
	queries := store.New(service.Pool)
	names, err := queries.DivisionNames(ctx, canonical)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDivisionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read division: %w", err)
	}
	state, err := service.loadLandslides(ctx)
	if err != nil {
		return nil, err
	}
	division := &Division{ID: canonical, Name: names.Name, NameSi: names.NameSi, Lon: names.Lon, Lat: names.Lat,
		Report: state.meta, History: []HistoryEntry{}}
	if names.DistrictID != nil {
		division.DistrictID = *names.DistrictID
	}
	if names.DistrictName != nil {
		division.District = *names.DistrictName
	}
	if state.show {
		for _, feature := range state.divisions {
			if feature.ID == canonical {
				words := feature.Properties.LevelWords
				division.Level = &words
			}
		}
	}
	history, err := queries.DivisionLandslideHistory(ctx, store.DivisionLandslideHistoryParams{DsID: canonical, MaxRows: historyRows})
	if err != nil {
		return nil, fmt.Errorf("read division history: %w", err)
	}
	for _, row := range history {
		division.History = append(division.History, HistoryEntry{Date: row.ReportDate.Time.Format(time.DateOnly), LevelWords: WordsFor(int(row.Level))})
	}
	return division, nil
}

// Filter narrows the bulletin list.
type Filter struct {
	// Type is a bulletin type, or "" for all.
	Type string
	// District is a district code, or "" for all.
	District string
	// Limit is the most bulletins to return; 0 is the default.
	Limit int
}

// ValidType reports whether a type can be filtered on.
func ValidType(kind string) bool {
	_, ok := TypeLabels[kind]
	return ok
}

// Overview is GET /v1/warnings.
type Overview struct {
	// Bulletins are the filtered list, newest first.
	Bulletins []Bulletin `json:"bulletins"`
	// InForce are warnings and advisories in force, one per warning, whatever
	// the filter.
	InForce []Bulletin `json:"inForce"`
	// ActiveCount is what the header's badge shows: warnings and advisories in
	// force, plus one if a landslide report is in force.
	ActiveCount int `json:"activeCount"`
	// Banner lists what warrants the prominent notice.
	Banner []BannerItem `json:"banner"`
	// Districts are the districts to highlight on the map, and why.
	Districts []DistrictHighlight `json:"districts"`
	Landslide LandslideMeta       `json:"landslide"`
	Meta      OverviewMeta        `json:"meta"`
}

// OverviewMeta is the list's origin and freshness.
type OverviewMeta struct {
	Source      string     `json:"source"`
	Attribution string     `json:"attribution"`
	Note        string     `json:"note"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	UpstreamAt  *time.Time `json:"upstreamAt"`
	GeneratedAt time.Time  `json:"generatedAt"`
	Stale       bool       `json:"stale"`
}

// BannerItem is one thing the banner announces, in the issuer's words.
type BannerItem struct {
	// Key identifies it; a dismissed key stays dismissed, a new one shows.
	Key string `json:"key"`
	// Kind is landslide or bulletin.
	Kind     string     `json:"kind"`
	Title    string     `json:"title"`
	Detail   string     `json:"detail"`
	Level    int        `json:"level,omitempty"`
	Colour   *string    `json:"colour"`
	IssuedAt *time.Time `json:"issuedAt"`
	PDFURL   *string    `json:"pdfUrl"`
}

// DistrictHighlight says why a district is highlighted. Only structured levels
// and exact words in clean text count.
type DistrictHighlight struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// LandslideLevel is the highest level of its divisions in the report shown, or 0.
	LandslideLevel int `json:"landslideLevel"`
	// Mentioned lists warnings and advisories in force that name it.
	Mentioned []Mention `json:"mentioned"`
}

// Mention is a bulletin naming a district.
type Mention struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Overview returns bulletins, what is in force, the banner and the highlights.
func (service *Service) Overview(ctx context.Context, filter Filter) (*Overview, error) {
	now := service.now()
	queries := store.New(service.Pool)
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	limit = min(limit, maxLimit)
	params := store.ListBulletinsParams{Since: now.Add(-listWindow), MaxRows: int32(limit)}
	if filter.Type != "" {
		params.BulletinType = &filter.Type
	}
	if filter.District != "" {
		params.District = &filter.District
	}
	rows, err := queries.ListBulletins(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list bulletins: %w", err)
	}
	inForce, err := service.inForce(ctx, queries, now)
	if err != nil {
		return nil, err
	}
	state, err := service.loadLandslides(ctx)
	if err != nil {
		return nil, err
	}

	overview := &Overview{Bulletins: toBulletins(rows, now), InForce: inForce, Landslide: state.meta, Meta: OverviewMeta{
		Source: dmcbulletins.SourceName, Attribution: dmcbulletins.Attribution, Note: bulletinNote, GeneratedAt: now,
	}}
	if version, err := queries.GetSourceVersion(ctx, dmcbulletins.SourceName); err == nil {
		fetched, upstream := version.FetchedAt, version.SourceTime
		overview.Meta.FetchedAt, overview.Meta.UpstreamAt = &fetched, &upstream
		overview.Meta.Stale = now.Sub(fetched) > service.BulletinStaleAfter || now.Sub(upstream) > service.BulletinStaleAfter
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("read bulletin version: %w", err)
	} else {
		overview.Meta.Stale = true
	}
	overview.ActiveCount = len(inForce)
	if state.show && len(state.divisions) > 0 {
		overview.ActiveCount++
	}
	overview.Banner = banner(state, inForce)
	overview.Districts = highlights(state, inForce)
	return overview, nil
}

func (service *Service) inForce(ctx context.Context, queries *store.Queries, now time.Time) ([]Bulletin, error) {
	rows, err := queries.ListBulletins(ctx, store.ListBulletinsParams{Since: now.Add(-inForceWindow), MaxRows: 400})
	if err != nil {
		return nil, fmt.Errorf("list bulletins in force: %w", err)
	}
	return InForce(toBulletins(rows, now)), nil
}

// banner picks what the prominent notice announces: a level 2 or 3 landslide
// warning on the map, and bulletins whose own words call them warnings.
// Advisories and routine reports do not raise it.
func banner(state loaded, inForce []Bulletin) []BannerItem {
	items := []BannerItem{}
	if state.show {
		counts := map[int]int{}
		for _, division := range state.divisions {
			counts[division.Properties.Level]++
		}
		if counts[2]+counts[3] > 0 {
			highest := 2
			detail := fmt.Sprintf("%s: %d DS division%s", WordsFor(2).Label, counts[2], plural(counts[2]))
			if counts[3] > 0 {
				highest = 3
				detail = fmt.Sprintf("%s: %d DS division%s", WordsFor(3).Label, counts[3], plural(counts[3]))
				if counts[2] > 0 {
					detail += fmt.Sprintf("; %s: %d", WordsFor(2).Label, counts[2])
				}
			}
			colour := WordsFor(highest).Colour
			date := ""
			if state.meta.ReportDate != nil {
				date = *state.meta.ReportDate
			}
			items = append(items, BannerItem{
				Key:  fmt.Sprintf("landslide:%s:%d:%d", date, counts[3], counts[2]),
				Kind: "landslide", Title: "Landslide early warning", Detail: detail, Level: highest, Colour: &colour,
				IssuedAt: state.meta.IssuedAt, PDFURL: state.meta.PDFURL,
			})
		}
	}
	for _, bulletin := range inForce {
		if bulletin.Severity != string(dmcbulletins.SeverityWarning) {
			continue
		}
		issued, pdf := bulletin.IssuedAt, bulletin.PDFURL
		items = append(items, BannerItem{Key: "bulletin:" + bulletin.ID, Kind: "bulletin", Title: bulletin.Title,
			Detail: bulletin.TypeLabel, Colour: bulletin.Colour, IssuedAt: &issued, PDFURL: &pdf})
	}
	return items
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func highlights(state loaded, inForce []Bulletin) []DistrictHighlight {
	byID := map[string]*DistrictHighlight{}
	get := func(id, name string) *DistrictHighlight {
		if entry, ok := byID[id]; ok {
			return entry
		}
		entry := &DistrictHighlight{ID: id, Name: name, Mentioned: []Mention{}}
		byID[id] = entry
		return entry
	}
	if state.show {
		for _, division := range state.divisions {
			entry := get(division.Properties.DistrictID, division.Properties.District)
			entry.LandslideLevel = max(entry.LandslideLevel, division.Properties.Level)
		}
	}
	for _, bulletin := range inForce {
		for _, district := range bulletin.Districts {
			entry := get(district.ID, district.Name)
			entry.Mentioned = append(entry.Mentioned, Mention{ID: bulletin.ID, Title: bulletin.Title})
		}
	}
	out := make([]DistrictHighlight, 0, len(byID))
	for _, entry := range byID {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func toBulletins(rows []store.OfficialBulletin, now time.Time) []Bulletin {
	out := make([]Bulletin, 0, len(rows))
	for _, row := range rows {
		districts := make([]DistrictRef, 0, len(row.Districts))
		for _, id := range row.Districts {
			districts = append(districts, DistrictRef{ID: id, Name: dmcbulletins.DistrictName(id)})
		}
		out = append(out, Bulletin{
			ID: row.DocID, Type: row.BulletinType, TypeLabel: TypeLabels[row.BulletinType], Title: row.Title,
			Severity: row.Severity, Colour: row.Colour, MessageID: row.MessageID, IssuedAt: row.IssuedAt,
			ValidFrom: row.ValidFrom, ValidUntil: row.ValidUntil,
			Current: IsCurrent(row.Severity, row.IssuedAt, row.ValidUntil, now),
			PDFURL:  row.PdfUrl, SourcePageURL: row.SourcePageUrl, Language: row.Lang,
			Text: row.Body, TextStatus: row.TextStatus, Districts: districts,
			Source: row.Source, SourceTime: row.SourceTime, FetchedAt: row.FetchedAt,
		})
	}
	return out
}

// LandslideHere is the landslide warning at a place.
type LandslideHere struct {
	// Status and Note speak about the latest report in general.
	Status string `json:"status"`
	Note   string `json:"note"`
	Stale  bool   `json:"stale"`
	// Level is the level of the DS division containing the place; null when
	// the division is not named or no report is in force.
	Level      *LevelWords `json:"level"`
	DSID       *string     `json:"dsId"`
	DS         *string     `json:"ds"`
	District   *string     `json:"district"`
	ReportDate *string     `json:"reportDate"`
	IssuedAt   *time.Time  `json:"issuedAt"`
	ValidUntil *time.Time  `json:"validUntil"`
	PDFURL     *string     `json:"pdfUrl"`
}

// OfficialAt is everything official about a place.
type OfficialAt struct {
	// Status is available, or not_available when nothing could be read.
	Status string `json:"status"`
	// Landslide is the warning for the place's DS division.
	Landslide *LandslideHere `json:"landslide"`
	// Warnings are recent bulletins naming the place's district by exact word.
	Warnings []Bulletin `json:"warnings"`
	// General are warnings in force everywhere and the latest of each type.
	General   []Bulletin `json:"general"`
	Note      string     `json:"note"`
	Stale     bool       `json:"stale"`
	FetchedAt *time.Time `json:"fetchedAt"`
}

// Official gathers what is official about a place in a district and DS
// division; either may be "" at sea.
func (service *Service) Official(ctx context.Context, districtID, dsID string) (*OfficialAt, error) {
	now := service.now()
	queries := store.New(service.Pool)
	state, err := service.loadLandslides(ctx)
	if err != nil {
		return nil, err
	}
	here := &LandslideHere{Status: state.meta.Status, Note: state.meta.Note, Stale: state.meta.Stale,
		ReportDate: state.meta.ReportDate, IssuedAt: state.meta.IssuedAt, ValidUntil: state.meta.ValidUntil, PDFURL: state.meta.PDFURL}
	if dsID != "" {
		id := dsID
		here.DSID = &id
		if names, err := queries.DivisionNames(ctx, dsID); err == nil {
			here.DS = &names.Name
			here.District = names.DistrictName
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("read division: %w", err)
		}
		if state.show {
			for _, division := range state.divisions {
				if division.ID == dsID {
					words := division.Properties.LevelWords
					here.Level = &words
				}
			}
		}
	}

	official := &OfficialAt{Status: "available", Landslide: here, Warnings: []Bulletin{}, General: []Bulletin{}, Note: bulletinNote}
	if version, err := queries.GetSourceVersion(ctx, dmcbulletins.SourceName); err == nil {
		fetched := version.FetchedAt
		official.FetchedAt = &fetched
		official.Stale = now.Sub(fetched) > service.BulletinStaleAfter || now.Sub(version.SourceTime) > service.BulletinStaleAfter
	} else if errors.Is(err, pgx.ErrNoRows) {
		official.Stale = true
	} else {
		return nil, fmt.Errorf("read bulletin version: %w", err)
	}

	if districtID != "" {
		rows, err := queries.ListBulletins(ctx, store.ListBulletinsParams{District: &districtID, Since: now.Add(-districtWindow), MaxRows: 10})
		if err != nil {
			return nil, fmt.Errorf("list bulletins for %s: %w", districtID, err)
		}
		official.Warnings = dedupe(toBulletins(rows, now))
	}
	named := map[string]bool{}
	for _, bulletin := range official.Warnings {
		named[bulletin.ID] = true
	}
	inForce, err := service.inForce(ctx, queries, now)
	if err != nil {
		return nil, err
	}
	for _, bulletin := range inForce {
		if !named[bulletin.ID] {
			official.General = append(official.General, bulletin)
			named[bulletin.ID] = true
		}
	}
	latest, err := queries.LatestBulletinOfEachType(ctx)
	if err != nil {
		return nil, fmt.Errorf("list latest bulletins: %w", err)
	}
	for _, bulletin := range toBulletins(latest, now) {
		// Landslide reports are spoken for by the slot above.
		if !named[bulletin.ID] && bulletin.Type != string(dmcbulletins.TypeLandslide) {
			official.General = append(official.General, bulletin)
			named[bulletin.ID] = true
		}
	}
	return official, nil
}

// dedupe keeps the newest bulletin of each title, for routine reports reissued
// through the day.
func dedupe(bulletins []Bulletin) []Bulletin {
	seen := map[string]bool{}
	out := make([]Bulletin, 0, len(bulletins))
	for _, bulletin := range bulletins {
		key := bulletin.Type + "|" + sameWarning(bulletin.Title)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, bulletin)
	}
	return out
}

func (service *Service) now() time.Time {
	if service.Now != nil {
		return service.Now().UTC()
	}
	return time.Now().UTC()
}
