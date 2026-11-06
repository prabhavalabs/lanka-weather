-- name: DeleteFloodGauges :exec
DELETE FROM flood_gauges;

-- name: InsertFloodGauge :exec
INSERT INTO flood_gauges (gauge_id, source, gauge_source, site_name, river, lon, lat, quality_verified, severity, trend,
                          issued_at, forecast_start, forecast_end, warning_level, danger_level, extreme_danger_level,
                          value_unit, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18);

-- name: ListFloodGauges :many
SELECT gauge_id, source, gauge_source, site_name, river, lon, lat, quality_verified, severity, trend, issued_at,
       forecast_start, forecast_end, warning_level, danger_level, extreme_danger_level, value_unit, fetched_at
FROM flood_gauges
ORDER BY gauge_id;

-- name: DeleteFloodAreasOfKinds :exec
-- The flash-flood collector replaces only flash-flood areas.
DELETE FROM flood_areas WHERE kind = ANY(@kinds::text[]);

-- name: DeleteFloodAreasNotOfKinds :exec
-- The forecast collector replaces every area but the flash floods.
DELETE FROM flood_areas WHERE kind <> ALL(@kinds::text[]);

-- name: ListFloodAreasSeen :many
-- When each stored area was first seen, with its shape, so a replacement
-- keeps the first sighting and need not read an unchanged polygon again.
SELECT id, first_seen_at, ST_AsGeoJSON(geom, 6)::text AS geojson
FROM flood_areas
WHERE kind = ANY(@kinds::text[]);

-- name: InsertFloodArea :exec
INSERT INTO flood_areas (id, kind, source, gauge_id, severity, issued_at, valid_from, valid_until, properties, geom,
                         fetched_at, first_seen_at)
VALUES (@id, @kind, @source, @gauge_id, @severity, @issued_at, @valid_from, @valid_until, @properties,
        ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_SetSRID(ST_GeomFromGeoJSON(@geojson::text), 4326)), 3)),
        @fetched_at, @first_seen_at)
ON CONFLICT (id) DO NOTHING;

-- name: ListFloodAreas :many
SELECT id, kind, source, gauge_id, severity, issued_at, valid_from, valid_until, properties,
       ST_AsGeoJSON(geom, 5)::text AS geojson, fetched_at, first_seen_at
FROM flood_areas
ORDER BY kind, id;

-- name: FloodAreasAt :many
-- Flood areas fetched since a time that contain a point, leaving out those of
-- gauges Google has not verified.
SELECT a.id, a.kind, a.gauge_id, a.severity, a.valid_from, a.valid_until
FROM flood_areas a
LEFT JOIN flood_gauges g ON g.gauge_id = a.gauge_id
WHERE ST_Intersects(a.geom, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326))
  AND a.fetched_at >= @since
  AND (a.gauge_id IS NULL OR g.quality_verified IS TRUE)
ORDER BY a.kind, a.id;

-- name: DeleteFloodGaugesFetchedBefore :exec
DELETE FROM flood_gauges WHERE fetched_at < $1;

-- name: DeleteFloodAreasFetchedBefore :exec
DELETE FROM flood_areas WHERE fetched_at < $1;
