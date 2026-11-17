-- name: UpsertBulletin :exec
-- Stores what the index says about a bulletin. Text and what was read from it
-- are left alone once fetched; severity follows the title until the text
-- gives a colour.
INSERT INTO official_bulletins (doc_id, bulletin_type, title, severity, issued_at, pdf_url, source_page_url, lang,
                                text_status, source, source_time, fetched_at)
VALUES (@doc_id, @bulletin_type, @title, @severity, @issued_at, @pdf_url, @source_page_url, @lang, 'pending', @source,
        @source_time, @fetched_at)
ON CONFLICT (doc_id) DO UPDATE
SET bulletin_type = EXCLUDED.bulletin_type,
    title = EXCLUDED.title,
    severity = CASE WHEN official_bulletins.colour IS NULL THEN EXCLUDED.severity ELSE official_bulletins.severity END,
    issued_at = EXCLUDED.issued_at,
    pdf_url = EXCLUDED.pdf_url,
    source_page_url = EXCLUDED.source_page_url,
    lang = EXCLUDED.lang,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: BulletinsNeedingText :many
-- Recent bulletins whose text has not been read yet, newest first, and any
-- withdrawal notice not yet read for the basins it lists, whenever it came.
SELECT doc_id, bulletin_type, title, severity, issued_at
FROM official_bulletins
WHERE (text_status = 'pending' OR (severity = 'withdrawal' AND withdrawn_basins IS NULL AND text_status <> 'unavailable'))
  AND issued_at >= @since
ORDER BY issued_at DESC, doc_id
LIMIT @max_rows;

-- name: UpdateBulletinText :exec
UPDATE official_bulletins
SET severity = @severity,
    colour = @colour,
    message_id = @message_id,
    valid_from = @valid_from,
    valid_until = @valid_until,
    body = @body,
    text_status = @text_status,
    districts = @districts,
    withdrawn_basins = @withdrawn_basins,
    text_fetched_at = @text_fetched_at
WHERE doc_id = @doc_id;

-- name: MarkBulletinTextUnavailable :exec
UPDATE official_bulletins
SET text_status = 'unavailable', text_fetched_at = @text_fetched_at
WHERE doc_id = @doc_id;

-- name: ListBulletins :many
-- Bulletins newest first, optionally of one type or naming one district.
SELECT *
FROM official_bulletins
WHERE (sqlc.narg('bulletin_type')::text IS NULL OR bulletin_type = sqlc.narg('bulletin_type'))
  AND (sqlc.narg('district')::text IS NULL OR sqlc.narg('district')::text = ANY (districts))
  AND issued_at >= @since
ORDER BY issued_at DESC, doc_id
LIMIT @max_rows;

-- name: ListWithdrawalNotices :many
-- Withdrawal notices, oldest first: the first after a warning is what ended it.
SELECT doc_id, issued_at, pdf_url, withdrawn_basins
FROM official_bulletins
WHERE severity = 'withdrawal'
  AND issued_at >= @since
ORDER BY issued_at, doc_id;

-- name: LatestBulletinOfEachType :many
-- The newest English bulletin of each type: the latest general bulletins.
SELECT DISTINCT ON (bulletin_type) *
FROM official_bulletins
WHERE lang <> 'ta'
ORDER BY bulletin_type, issued_at DESC, doc_id;

-- name: LatestLandslideBulletin :one
-- The newest landslide bulletin whatever its date.
SELECT *
FROM official_bulletins
WHERE bulletin_type = 'landslide'
ORDER BY issued_at DESC, doc_id
LIMIT 1;

-- name: LandslideBulletinsBetween :many
-- Landslide bulletins issued in a span, newest first, to match them to
-- reports by their local date.
SELECT *
FROM official_bulletins
WHERE bulletin_type = 'landslide'
  AND issued_at >= @since
ORDER BY issued_at DESC, doc_id;

-- name: UpsertLandslideReport :exec
INSERT INTO landslide_reports (report_date, source, source_time, fetched_at)
VALUES (@report_date, @source, @source_time, @fetched_at)
ON CONFLICT (report_date) DO UPDATE
SET source = EXCLUDED.source,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: DeleteLandslideLevels :exec
DELETE FROM landslide_levels WHERE report_date = @report_date;

-- name: InsertLandslideLevels :copyfrom
INSERT INTO landslide_levels (report_date, ds_id, district_id, level)
VALUES ($1, $2, $3, $4);

-- name: LatestLandslideReport :one
SELECT report_date, source, source_time, fetched_at
FROM landslide_reports
ORDER BY report_date DESC
LIMIT 1;

-- name: LandslideReportLevels :many
-- One report's divisions with their names, label points and boundaries. The
-- region tiles carry DS divisions only from zoom 8, so the layer draws these
-- boundaries itself and is visible at country scale. A code that matches no
-- region comes back without a name, so it can be reported.
SELECT l.ds_id, l.district_id, l.level,
       r.name AS ds_name, r.name_si AS ds_name_si, d.name AS district_name,
       COALESCE(ST_X(r.label_point), 0)::float8 AS lon, COALESCE(ST_Y(r.label_point), 0)::float8 AS lat,
       COALESCE(ST_AsGeoJSON(r.geom_simple, 4), '')::text AS geojson
FROM landslide_levels l
LEFT JOIN regions r ON r.id = l.ds_id
LEFT JOIN regions d ON d.id = l.district_id
WHERE l.report_date = @report_date
ORDER BY l.level DESC, l.ds_id;

-- name: DivisionLandslideHistory :many
-- A division's level in each report that named it, newest first.
SELECT l.report_date, l.level
FROM landslide_levels l
WHERE l.ds_id = @ds_id
ORDER BY l.report_date DESC
LIMIT @max_rows;

-- name: DivisionLevelInReport :one
SELECT level FROM landslide_levels WHERE report_date = @report_date AND ds_id = @ds_id;

-- name: DivisionNames :one
SELECT r.id, r.name, r.name_si, r.district_id, d.name AS district_name,
       ST_X(r.label_point)::float8 AS lon, ST_Y(r.label_point)::float8 AS lat
FROM regions r
LEFT JOIN regions d ON d.id = r.district_id
WHERE r.id = @id;

-- name: GetBulletin :one
SELECT *
FROM official_bulletins
WHERE doc_id = @doc_id;
