-- name: UpsertRiverStation :exec
INSERT INTO river_stations (id, name, basin, lon, lat, unit, alert_level, minor_flood_level,
                            major_flood_level, source, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    basin = EXCLUDED.basin,
    lon = EXCLUDED.lon,
    lat = EXCLUDED.lat,
    unit = EXCLUDED.unit,
    alert_level = EXCLUDED.alert_level,
    minor_flood_level = EXCLUDED.minor_flood_level,
    major_flood_level = EXCLUDED.major_flood_level,
    source = EXCLUDED.source,
    source_url = NULL,
    upstream = NULL,
    updated_at = EXCLUDED.updated_at;

-- name: InsertRiverReading :batchexec
INSERT INTO river_readings (station_id, observed_at, water_level, rainfall_mm, fetched_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (station_id, observed_at) DO UPDATE
SET water_level = EXCLUDED.water_level,
    rainfall_mm = EXCLUDED.rainfall_mm,
    fetched_at = EXCLUDED.fetched_at;

-- name: LatestRiverObservation :one
-- The newest reading of one collector's stations: stations from another
-- collector must not move the point a collector resumes from.
SELECT r.observed_at
FROM river_readings r
JOIN river_stations s ON s.id = r.station_id
WHERE s.source = $1
ORDER BY r.observed_at DESC
LIMIT 1;

-- name: ListRiverStations :many
SELECT id, name, basin, lon, lat, unit, alert_level, minor_flood_level, major_flood_level, updated_at,
       source, source_url, upstream
FROM river_stations
ORDER BY basin, name;

-- name: GetRiverStation :one
SELECT id, name, basin, lon, lat, unit, alert_level, minor_flood_level, major_flood_level, updated_at,
       source, source_url, upstream
FROM river_stations
WHERE id = $1;

-- name: RecentRiverReadings :many
-- Readings for every station since a time, newest first per station; used to
-- derive each station's current level and trend.
SELECT station_id, observed_at, water_level, rainfall_mm
FROM river_readings
WHERE observed_at >= $1
ORDER BY station_id, observed_at DESC;

-- name: StationRiverReadings :many
SELECT observed_at, water_level, rainfall_mm
FROM river_readings
WHERE station_id = $1 AND observed_at >= $2
ORDER BY observed_at;

-- name: DeleteRiverReadingsBefore :exec
DELETE FROM river_readings WHERE observed_at < $1;

-- name: UpsertExtraRiverStation :exec
-- A station from a secondary collector. It never takes over a station of
-- another source, and a report without a reading keeps the last known unit and
-- thresholds, because a quiet gauge publishes neither.
INSERT INTO river_stations (id, name, basin, lon, lat, unit, alert_level, minor_flood_level,
                            major_flood_level, source, source_url, upstream, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    basin = EXCLUDED.basin,
    lon = EXCLUDED.lon,
    lat = EXCLUDED.lat,
    unit = CASE WHEN EXCLUDED.unit = 'unknown' THEN river_stations.unit ELSE EXCLUDED.unit END,
    alert_level = CASE WHEN EXCLUDED.unit = 'unknown' THEN river_stations.alert_level ELSE EXCLUDED.alert_level END,
    minor_flood_level = CASE WHEN EXCLUDED.unit = 'unknown' THEN river_stations.minor_flood_level ELSE EXCLUDED.minor_flood_level END,
    major_flood_level = CASE WHEN EXCLUDED.unit = 'unknown' THEN river_stations.major_flood_level ELSE EXCLUDED.major_flood_level END,
    source_url = EXCLUDED.source_url,
    upstream = EXCLUDED.upstream,
    updated_at = EXCLUDED.updated_at
WHERE river_stations.source = EXCLUDED.source;

-- name: RiverStationSources :many
-- Who owns each station, and where it is, to tell which reports are new gauges.
SELECT id, name, source, lon, lat FROM river_stations;

-- name: InsertNextHour :batchexec
INSERT INTO river_next_hour (station_id, issued_at, level_ft, low_ft, high_ft, change_ft,
                             direction, crossing, method, source, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (station_id, issued_at) DO UPDATE
SET level_ft = EXCLUDED.level_ft,
    low_ft = EXCLUDED.low_ft,
    high_ft = EXCLUDED.high_ft,
    change_ft = EXCLUDED.change_ft,
    direction = EXCLUDED.direction,
    crossing = EXCLUDED.crossing,
    method = EXCLUDED.method,
    source = EXCLUDED.source,
    fetched_at = EXCLUDED.fetched_at;

-- name: LatestNextHour :many
-- Each station's newest forecast issued since a time.
SELECT DISTINCT ON (station_id) station_id, issued_at, level_ft, low_ft, high_ft, change_ft,
       direction, crossing, method, source, fetched_at
FROM river_next_hour
WHERE issued_at >= $1
ORDER BY station_id, issued_at DESC;

-- name: DeleteNextHourBefore :exec
DELETE FROM river_next_hour WHERE issued_at < $1;
