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
    updated_at = EXCLUDED.updated_at;

-- name: InsertRiverReading :batchexec
INSERT INTO river_readings (station_id, observed_at, water_level, rainfall_mm, fetched_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (station_id, observed_at) DO UPDATE
SET water_level = EXCLUDED.water_level,
    rainfall_mm = EXCLUDED.rainfall_mm,
    fetched_at = EXCLUDED.fetched_at;

-- name: LatestRiverObservation :one
SELECT observed_at FROM river_readings ORDER BY observed_at DESC LIMIT 1;

-- name: ListRiverStations :many
SELECT id, name, basin, lon, lat, unit, alert_level, minor_flood_level, major_flood_level, updated_at
FROM river_stations
ORDER BY basin, name;

-- name: GetRiverStation :one
SELECT id, name, basin, lon, lat, unit, alert_level, minor_flood_level, major_flood_level, updated_at
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
