-- name: UpsertRainGauge :exec
-- The newest reading only ever moves forward, so a feed replaying old points
-- cannot make a gauge look older or newer than it is.
INSERT INTO rain_gauges (id, device, name, institution, lon, lat, latest_observed_at, latest_rain_mm, source, source_time, fetched_at)
VALUES (@id, @device, @name, @institution, @lon, @lat, @latest_observed_at, @latest_rain_mm, @source, @source_time, @fetched_at)
ON CONFLICT (id) DO UPDATE
SET device = EXCLUDED.device,
    name = EXCLUDED.name,
    institution = EXCLUDED.institution,
    lon = EXCLUDED.lon,
    lat = EXCLUDED.lat,
    latest_observed_at = CASE WHEN rain_gauges.latest_observed_at IS NULL OR EXCLUDED.latest_observed_at >= rain_gauges.latest_observed_at
                              THEN EXCLUDED.latest_observed_at ELSE rain_gauges.latest_observed_at END,
    latest_rain_mm = CASE WHEN rain_gauges.latest_observed_at IS NULL OR EXCLUDED.latest_observed_at >= rain_gauges.latest_observed_at
                          THEN EXCLUDED.latest_rain_mm ELSE rain_gauges.latest_rain_mm END,
    source = EXCLUDED.source,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: InsertRainGaugeReading :batchexec
INSERT INTO rain_gauge_readings (gauge_id, observed_at, rain_mm, source, source_time, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (gauge_id, observed_at) DO UPDATE
SET rain_mm = EXCLUDED.rain_mm,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: ListRainGauges :many
SELECT id, device, name, institution, lon, lat, latest_observed_at, latest_rain_mm, source, source_time, fetched_at
FROM rain_gauges
ORDER BY id;

-- name: GetRainGauge :one
SELECT id, device, name, institution, lon, lat, latest_observed_at, latest_rain_mm, source, source_time, fetched_at
FROM rain_gauges
WHERE id = $1;

-- name: RainGaugeReadingsSince :many
SELECT observed_at, rain_mm
FROM rain_gauge_readings
WHERE gauge_id = $1 AND observed_at >= $2
ORDER BY observed_at;

-- name: DeleteRainGaugeReadingsBefore :exec
DELETE FROM rain_gauge_readings WHERE observed_at < $1;

-- name: DeleteRainGaugesFetchedBefore :exec
DELETE FROM rain_gauges WHERE fetched_at < $1;

-- name: RainGaugeReadingsAllSince :many
-- Every gauge's readings since a time, for the rain the alert engine measures.
SELECT gauge_id, observed_at, rain_mm
FROM rain_gauge_readings
WHERE observed_at >= $1
ORDER BY gauge_id, observed_at;
