package dmcbulletins

import (
	"context"
	"crypto/md5" //nolint:gosec // reproduces the upstream index's identifiers, not for security
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prabhavalabs/lanka-weather/internal/source"
	"github.com/prabhavalabs/lanka-weather/internal/store"
)

// SiteSourceName identifies the DMC website's report lists, read directly.
const SiteSourceName = "dmc-site"

// SiteInterval is how often the DMC's report lists are read. The index on
// GitHub rebuilds about every two hours, which held official flood warnings
// back by 80 to 170 minutes (November 2026); the lists themselves are small
// pages, so reading them every ten minutes costs the DMC little.
const SiteInterval = 10 * time.Minute

const (
	defaultSiteBase = "https://www.dmc.gov.lk"
	// siteTextWindow is how old a new bulletin may be for its PDF to be read
	// here rather than left to the index's text.
	siteTextWindow = 2 * 24 * time.Hour
	// maxPDFsPerRun bounds the PDFs read in one run.
	maxPDFsPerRun = 6
	// maxPDFBytes caps one PDF; DMC bulletins are under 3 MB.
	maxPDFBytes = 16 << 20
)

// sitePages are each family's report list on dmc.gov.lk, as nuuuwan/lk_dmc
// reads them.
var sitePages = map[Type]struct{ itemID, reportType string }{
	TypeSituation:  {"273", "1"},
	TypeWeather:    {"274", "2"},
	TypeLandslide:  {"276", "5"},
	TypeWaterLevel: {"277", "6"},
}

// SitePageURL is the first page of a family's report list.
func SitePageURL(base string, kind Type) string {
	page := sitePages[kind]
	query := url.Values{"option": {"com_dmcreports"}, "view": {"reports"}, "Itemid": {page.itemID},
		"report_type_id": {page.reportType}, "lang": {"en"}, "limitstart": {"0"}}
	return base + "/index.php?" + query.Encode()
}

// TextExtractor turns a PDF into plain text.
type TextExtractor interface {
	Text(ctx context.Context, pdf []byte) (string, error)
}

// SiteCollector reads the newest ten reports of each family from the DMC's
// own lists, so a warning is stored minutes after it is posted rather than
// when the index on GitHub next rebuilds. Rows share the index's identifiers,
// so the two sources fill the same bulletin. The PDF of a new warning,
// advisory, withdrawal or landslide report is read here when Text is set;
// otherwise its text comes from the index as before.
type SiteCollector struct {
	Pool    *pgxpool.Pool
	Fetcher source.Fetcher
	// Text extracts a PDF's text; nil leaves text to the index.
	Text   TextExtractor
	Logger *slog.Logger
	// Base overrides the website's address, for tests.
	Base string
}

// Name implements source.Collector.
func (collector *SiteCollector) Name() string { return SiteSourceName }

// Interval implements source.Collector.
func (collector *SiteCollector) Interval() time.Duration { return SiteInterval }

// RetryAfter implements source.Retrier: a failed run is tried again at the
// next turn.
func (collector *SiteCollector) RetryAfter() time.Duration { return SiteInterval }

func (collector *SiteCollector) logger() *slog.Logger {
	if collector.Logger != nil {
		return collector.Logger
	}
	return slog.Default()
}

// Collect reads each family's list, stores the rows and reads new PDFs.
func (collector *SiteCollector) Collect(ctx context.Context, now time.Time) (int, error) {
	base := collector.Base
	if base == "" {
		base = defaultSiteBase
	}
	queries := store.New(collector.Pool)
	stored := 0
	var failures []error
	for _, kind := range Types {
		pageURL := SitePageURL(base, kind)
		page, err := collector.Fetcher.Fetch(ctx, source.Request{URL: pageURL, Accept: "text/html", MaxBytes: 2 << 20})
		if err != nil {
			failures = append(failures, fmt.Errorf("%s list: %w", kind, err))
			continue
		}
		bulletins, err := DecodeSitePage(page, kind, base, pageURL)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s list: %w", kind, err))
			continue
		}
		for _, bulletin := range bulletins {
			if err := queries.UpsertBulletin(ctx, store.UpsertBulletinParams{
				DocID: bulletin.DocID, BulletinType: string(bulletin.Type), Title: bulletin.Title,
				Severity: string(bulletin.Severity), IssuedAt: bulletin.IssuedAt, PdfUrl: bulletin.PDFURL,
				SourcePageUrl: bulletin.SourcePageURL, Lang: bulletin.Language, Source: SiteSourceName,
				SourceTime: now, FetchedAt: now,
			}); err != nil {
				failures = append(failures, fmt.Errorf("upsert %s: %w", bulletin.DocID, err))
				continue
			}
			stored++
		}
	}
	if collector.Text != nil {
		if err := collector.readPDFs(ctx, queries, now); err != nil {
			failures = append(failures, err)
		}
	}
	if stored > 0 {
		if err := queries.PutSourceVersion(ctx, store.PutSourceVersionParams{
			Source: SiteSourceName, Version: now.UTC().Format(time.RFC3339), SourceTime: now, FetchedAt: now, Records: int32(stored),
		}); err != nil {
			failures = append(failures, err)
		}
	}
	return stored, errors.Join(failures...)
}

// readPDFs reads the text of new bulletins that matter for alerts: routine
// water level and weather reports are left to the index.
func (collector *SiteCollector) readPDFs(ctx context.Context, queries *store.Queries, now time.Time) error {
	rows, err := queries.BulletinsNeedingText(ctx, store.BulletinsNeedingTextParams{Since: now.Add(-siteTextWindow), MaxRows: 40})
	if err != nil {
		return fmt.Errorf("list bulletins needing text: %w", err)
	}
	var failures []error
	read := 0
	for _, row := range rows {
		kind := Type(row.BulletinType)
		if row.Severity == string(SeverityRoutine) && kind != TypeLandslide {
			continue
		}
		if read >= maxPDFsPerRun {
			break
		}
		bulletin, err := queries.GetBulletin(ctx, row.DocID)
		if err != nil {
			failures = append(failures, fmt.Errorf("read %s: %w", row.DocID, err))
			continue
		}
		if !strings.HasPrefix(bulletin.PdfUrl, defaultSiteBase+"/") && !strings.HasPrefix(bulletin.PdfUrl, collector.Base+"/") {
			continue
		}
		read++
		pdf, err := collector.Fetcher.Fetch(ctx, source.Request{URL: bulletin.PdfUrl, Accept: "application/pdf", MaxBytes: maxPDFBytes,
			Timeout: 90 * time.Second})
		if err != nil {
			failures = append(failures, fmt.Errorf("fetch pdf of %s: %w", row.DocID, err))
			continue
		}
		text, err := collector.Text.Text(ctx, pdf)
		if err != nil {
			failures = append(failures, fmt.Errorf("read pdf of %s: %w", row.DocID, err))
			continue
		}
		extract := ExtractText(JoinPDFBlocks(text), kind, row.IssuedAt)
		if extract.Status != TextClean && kind != TypeLandslide && row.Severity != string(SeverityWithdrawal) {
			// Leave a text that reads badly here to the index, which may read it better.
			continue
		}
		if err := queries.UpdateBulletinText(ctx, store.UpdateBulletinTextParams{
			DocID: row.DocID, Severity: string(Classify(kind, row.Title, extract.Colour)),
			Colour: optional(extract.Colour), MessageID: optional(extract.MessageID),
			ValidFrom: extract.ValidFrom, ValidUntil: extract.ValidUntil, Body: optional(extract.Body),
			TextStatus: extract.Status, Districts: nonNil(extract.Districts), TextFetchedAt: &now,
			WithdrawnBasins: withdrawnBasins(row.Severity, extract),
		}); err != nil {
			failures = append(failures, fmt.Errorf("store text of %s: %w", row.DocID, err))
			continue
		}
		collector.logger().Info("dmc bulletin read from site", "doc", row.DocID, "type", kind,
			"delay_minutes", int(now.Sub(row.IssuedAt).Minutes()), "districts", len(extract.Districts))
	}
	return errors.Join(failures...)
}

// JoinPDFBlocks writes pdftotext's output as the index's text is written: one
// paragraph per block, blocks apart by a blank line.
func JoinPDFBlocks(raw string) string {
	raw = strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\f", "\n\n")
	blocks := []string{}
	for _, block := range strings.Split(raw, "\n\n") {
		if joined := strings.Join(strings.Fields(block), " "); joined != "" {
			blocks = append(blocks, joined)
		}
	}
	return strings.Join(blocks, "\n\n")
}

var (
	siteRow   = regexp.MustCompile(`(?is)<tr[^>]*class="rowdate_\d+"[^>]*>(.*?)</tr>`)
	siteCell  = regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	siteHref  = regexp.MustCompile(`(?is)href="([^"]+)"`)
	siteTags  = regexp.MustCompile(`(?s)<[^>]*>`)
	siteDate  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	siteClock = regexp.MustCompile(`^\d{2}:\d{2}$`)
	// notWord is what the index drops from a title to name a document.
	notWord = regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	notID   = regexp.MustCompile(`[^a-zA-Z0-9\-]`)
)

// DecodeSitePage reads a DMC report list: title, date, time and the PDF link
// of each row. A row that is not a PDF (some reports are images) is skipped.
// A page with no table at all is an error, so a changed site is noticed.
func DecodeSitePage(page []byte, kind Type, base, pageURL string) ([]Bulletin, error) {
	text := string(page)
	if !strings.Contains(text, "mtable") {
		return nil, errors.New("page has no report table")
	}
	bulletins := []Bulletin{}
	for _, row := range siteRow.FindAllStringSubmatch(text, -1) {
		cells := siteCell.FindAllStringSubmatch(row[1], -1)
		if len(cells) != 4 {
			continue
		}
		cell := func(index int) string {
			return strings.Join(strings.Fields(html.UnescapeString(siteTags.ReplaceAllString(cells[index][1], " "))), " ")
		}
		title, date, clock := cell(0), cell(1), cell(2)
		href := siteHref.FindStringSubmatch(cells[3][1])
		if title == "" || !siteDate.MatchString(date) || !siteClock.MatchString(clock) || href == nil {
			continue
		}
		link := html.UnescapeString(href[1])
		if !strings.HasSuffix(strings.ToLower(link), ".pdf") {
			continue
		}
		if strings.HasPrefix(link, "/") {
			link = base + link
		}
		if !httpLink(link) {
			continue
		}
		issued, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, sriLanka)
		if err != nil {
			continue
		}
		bulletins = append(bulletins, Bulletin{
			DocID: SiteDocID(date, clock, title), Type: kind, Title: title, Severity: Classify(kind, title, ""),
			IssuedAt: issued.UTC(), PDFURL: link, SourcePageURL: pageURL, Language: language(title, "en"),
		})
	}
	return bulletins, nil
}

// SiteDocID is the identifier nuuuwan/lk_dmc gives a report, so a bulletin
// read here and the same one in the index are one row: the time and the
// title's words as a slug, shortened with an MD5 when long.
func SiteDocID(date, clock, title string) string {
	cleaned := strings.ReplaceAll(strings.TrimSpace(notWord.ReplaceAllString(strings.Join(strings.Fields(title), " "), "")), " ", "-")
	num := strings.ReplaceAll(clock, ":", "-") + "-" + strings.ToLower(cleaned)
	if len(num) >= 32 {
		sum := md5.Sum([]byte(num)) //nolint:gosec // see the import
		num = num[:23] + "-" + hex.EncodeToString(sum[:])[:8]
	}
	return notID.ReplaceAllString(date+"-"+num, "-")
}
