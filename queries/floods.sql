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

-- name: DeleteFloodAreas :exec
DELETE FROM flood_areas;

-- name: InsertFloodArea :exec
INSERT INTO flood_areas (id, kind, source, gauge_id, severity, issued_at, valid_from, valid_until, properties, geom,
                         fetched_at)
VALUES (@id, @kind, @source, @gauge_id, @severity, @issued_at, @valid_from, @valid_until, @properties,
        ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_SetSRID(ST_GeomFromGeoJSON(@geojson::text), 4326)), 3)),
        @fetched_at)
ON CONFLICT (id) DO NOTHING;

-- name: ListFloodAreas :many
SELECT id, kind, source, gauge_id, severity, issued_at, valid_from, valid_until, properties,
       ST_AsGeoJSON(geom, 5)::text AS geojson, fetched_at
FROM flood_areas
ORDER BY kind, id;

-- name: FloodAreasAt :many
-- Flood areas fetched since a time that contain a point.
SELECT id, kind, gauge_id, severity, valid_from, valid_until
FROM flood_areas
WHERE ST_Intersects(geom, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326))
  AND fetched_at >= @since
ORDER BY kind, id;

-- name: DeleteFloodGaugesFetchedBefore :exec
DELETE FROM flood_gauges WHERE fetched_at < $1;

-- name: DeleteFloodAreasFetchedBefore :exec
DELETE FROM flood_areas WHERE fetched_at < $1;
