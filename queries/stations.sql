-- name: UpsertMetStation :batchexec
INSERT INTO met_stations (id, name, wmo_id, lon, lat, elevation_m, position_source, source, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    wmo_id = EXCLUDED.wmo_id,
    lon = EXCLUDED.lon,
    lat = EXCLUDED.lat,
    elevation_m = EXCLUDED.elevation_m,
    position_source = EXCLUDED.position_source,
    source = EXCLUDED.source,
    updated_at = EXCLUDED.updated_at;

-- name: InsertMetReading3h :batchexec
INSERT INTO met_readings_3h (station_id, observed_at, rain_mm, temp_c, humidity_pct, source, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (station_id, observed_at) DO UPDATE
SET rain_mm = EXCLUDED.rain_mm,
    temp_c = EXCLUDED.temp_c,
    humidity_pct = EXCLUDED.humidity_pct,
    source = EXCLUDED.source,
    fetched_at = EXCLUDED.fetched_at;

-- name: InsertMetReadingDaily :batchexec
INSERT INTO met_readings_daily (station_id, report_date, observed_at, rain_mm, temp_min_c, temp_max_c, source, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (station_id, report_date) DO UPDATE
SET observed_at = EXCLUDED.observed_at,
    rain_mm = EXCLUDED.rain_mm,
    temp_min_c = EXCLUDED.temp_min_c,
    temp_max_c = EXCLUDED.temp_max_c,
    source = EXCLUDED.source,
    fetched_at = EXCLUDED.fetched_at;

-- name: CountMetReadings3h :one
SELECT count(*)::bigint FROM met_readings_3h;

-- name: ListMetStations :many
SELECT id, name, wmo_id, lon, lat, elevation_m, position_source, source, updated_at
FROM met_stations
ORDER BY name;

-- name: GetMetStation :one
SELECT id, name, wmo_id, lon, lat, elevation_m, position_source, source, updated_at
FROM met_stations
WHERE id = $1;

-- name: LatestMetReadings3h :many
-- Every station's readings in the day before its own newest reading, newest
-- first, so a source that has stopped still gives each station a last value.
SELECT r.station_id, r.observed_at, r.rain_mm, r.temp_c, r.humidity_pct, r.source, r.fetched_at
FROM met_readings_3h AS r
JOIN (SELECT station_id, max(observed_at) AS newest FROM met_readings_3h GROUP BY station_id) AS latest
  ON latest.station_id = r.station_id
WHERE r.observed_at > latest.newest - interval '24 hours'
ORDER BY r.station_id, r.observed_at DESC;

-- name: LatestMetReadingsDaily :many
-- Every station's newest daily report.
SELECT DISTINCT ON (station_id) station_id, report_date, observed_at, rain_mm, temp_min_c, temp_max_c, source, fetched_at
FROM met_readings_daily
ORDER BY station_id, report_date DESC;

-- name: StationMetReadings3h :many
SELECT observed_at, rain_mm, temp_c, humidity_pct
FROM met_readings_3h
WHERE station_id = $1 AND observed_at >= $2
ORDER BY observed_at;

-- name: StationMetReadingsDaily :many
SELECT report_date, observed_at, rain_mm, temp_min_c, temp_max_c
FROM met_readings_daily
WHERE station_id = $1 AND observed_at >= $2
ORDER BY observed_at;

-- name: MetFeedFreshness :one
-- The newest observation and fetch of each feed; the Unix epoch before its
-- first run.
SELECT COALESCE((SELECT max(observed_at) FROM met_readings_3h), 'epoch')::timestamptz AS newest_3h,
       COALESCE((SELECT max(fetched_at) FROM met_readings_3h), 'epoch')::timestamptz AS fetched_3h,
       COALESCE((SELECT max(observed_at) FROM met_readings_daily), 'epoch')::timestamptz AS newest_daily,
       COALESCE((SELECT max(fetched_at) FROM met_readings_daily), 'epoch')::timestamptz AS fetched_daily;

-- name: DeleteMetReadings3hBefore :exec
DELETE FROM met_readings_3h WHERE observed_at < $1;

-- name: DeleteMetReadingsDailyBefore :exec
DELETE FROM met_readings_daily WHERE observed_at < $1;
