-- name: UpsertReservoir :exec
-- Keeps the coordinates, which the map collector owns.
INSERT INTO reservoirs (id, name, district, river, kind, source, source_url, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    district = EXCLUDED.district,
    river = EXCLUDED.river,
    kind = EXCLUDED.kind,
    source = EXCLUDED.source,
    source_url = EXCLUDED.source_url,
    updated_at = EXCLUDED.updated_at;

-- name: UpsertReservoirReading :exec
INSERT INTO reservoir_readings (reservoir_id, snapshot_day, reading_day, status, storage_percent,
                                storage_fresh, flow_kind, flow_cusecs, flow_agency, flow_day,
                                agency, source, source_time, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (reservoir_id, snapshot_day) DO UPDATE
SET reading_day = EXCLUDED.reading_day,
    status = EXCLUDED.status,
    storage_percent = EXCLUDED.storage_percent,
    storage_fresh = EXCLUDED.storage_fresh,
    flow_kind = EXCLUDED.flow_kind,
    flow_cusecs = EXCLUDED.flow_cusecs,
    flow_agency = EXCLUDED.flow_agency,
    flow_day = EXCLUDED.flow_day,
    agency = EXCLUDED.agency,
    source = EXCLUDED.source,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: ListReservoirTargets :many
-- What the map collector needs to match a reservoir to OpenStreetMap.
SELECT id, name, district, kind FROM reservoirs ORDER BY id;

-- name: ClearReservoirCandidates :exec
TRUNCATE reservoir_candidate_staging;

-- name: StageReservoirCandidates :copyfrom
INSERT INTO reservoir_candidate_staging (osm_id, lines) VALUES ($1, $2);

-- name: MeasureReservoirCandidates :many
-- Builds each candidate's polygon and finds the districts it touches. A shape
-- that closes no ring (a dam drawn as a line) has no area and a point on the
-- line. Every district a water body touches counts, as many reservoirs lie on
-- a boundary. near_districts are those within about 15 km (0.135 degrees).
WITH shapes AS (
    SELECT s.osm_id, line.geom AS lines,
           ST_CollectionExtract(ST_MakeValid(ST_BuildArea(line.geom)), 3) AS polygon
    FROM reservoir_candidate_staging s
    CROSS JOIN LATERAL (SELECT ST_SetSRID(ST_GeomFromGeoJSON(s.lines), 4326) AS geom) line
),
measured AS (
    SELECT osm_id,
           CASE WHEN NOT ST_IsEmpty(polygon) THEN polygon END AS polygon,
           coalesce(CASE WHEN NOT ST_IsEmpty(polygon) THEN ST_PointOnSurface(polygon) END,
                    ST_ClosestPoint(lines, ST_Centroid(lines))) AS point,
           lines
    FROM shapes
)
SELECT m.osm_id,
       ST_X(m.point)::float8 AS lon,
       ST_Y(m.point)::float8 AS lat,
       (coalesce(ST_Area(m.polygon::geography), 0) / 10000)::float8 AS area_ha,
       coalesce((SELECT array_agg(r.name ORDER BY r.name)
                 FROM regions r
                 WHERE r.level = 'district'
                   AND ST_Intersects(r.geom, coalesce(m.polygon, m.point))), '{}')::text[] AS districts,
       coalesce((SELECT array_agg(r.name ORDER BY r.name)
                 FROM regions r
                 WHERE r.level = 'district'
                   AND ST_DWithin(r.geom_simple, coalesce(m.polygon, m.point), 0.135)), '{}')::text[] AS near_districts
FROM measured m;

-- name: ClearReservoirMap :exec
UPDATE reservoirs
SET lon = NULL, lat = NULL, area = NULL, osm_id = NULL, osm_name = NULL, osm_kind = NULL,
    map_source = @map_source::text, map_source_time = @map_source_time::timestamptz,
    map_fetched_at = @map_fetched_at::timestamptz;

-- name: SetReservoirMap :exec
-- Stores a match: the candidate's polygon for a water body, or only its point
-- for a dam.
UPDATE reservoirs r
SET lon = @lon::float8, lat = @lat::float8,
    area = CASE WHEN @osm_kind::text = 'water'
                THEN (SELECT ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_BuildArea(
                                 ST_SetSRID(ST_GeomFromGeoJSON(s.lines), 4326))), 3))
                      FROM reservoir_candidate_staging s WHERE s.osm_id = @osm_id::text)
           END,
    osm_id = @osm_id::text, osm_name = @osm_name::text, osm_kind = @osm_kind::text
WHERE r.id = @id::text;

-- name: LatestReservoirs :many
-- Every reservoir with its newest snapshot, for the map layer and the lists.
SELECT r.id, r.name, r.district, r.river, r.kind, r.source, r.source_url, r.lon, r.lat,
       r.osm_id, r.osm_name, r.osm_kind, r.map_source_time, r.map_fetched_at,
       (r.area IS NOT NULL)::bool AS has_area,
       l.snapshot_day, l.reading_day, l.status, l.storage_percent, l.storage_fresh,
       l.flow_kind, l.flow_cusecs, l.flow_agency, l.flow_day, l.agency, l.source_time, l.fetched_at
FROM reservoirs r
JOIN LATERAL (
    SELECT * FROM reservoir_readings x
    WHERE x.reservoir_id = r.id
    ORDER BY x.snapshot_day DESC
    LIMIT 1
) l ON true
ORDER BY r.name, r.id;

-- name: GetReservoir :one
SELECT r.id, r.name, r.district, r.river, r.kind, r.source, r.source_url, r.lon, r.lat,
       r.osm_id, r.osm_name, r.osm_kind, r.map_source_time, r.map_fetched_at,
       (r.area IS NOT NULL)::bool AS has_area,
       l.snapshot_day, l.reading_day, l.status, l.storage_percent, l.storage_fresh,
       l.flow_kind, l.flow_cusecs, l.flow_agency, l.flow_day, l.agency, l.source_time, l.fetched_at
FROM reservoirs r
JOIN LATERAL (
    SELECT * FROM reservoir_readings x
    WHERE x.reservoir_id = r.id
    ORDER BY x.snapshot_day DESC
    LIMIT 1
) l ON true
WHERE r.id = $1;

-- name: ReservoirAreas :many
-- Simplified to about 30 m: enough for the map, a fraction of the size.
SELECT r.id,
       ST_AsGeoJSON(ST_SimplifyPreserveTopology(r.area, 0.0003), 5)::text AS geometry
FROM reservoirs r
WHERE r.area IS NOT NULL
ORDER BY r.id;

-- name: ReservoirHistory :many
-- One row per day read, newest snapshot first within a day, oldest day first.
-- Days the source only repeated an old figure are left out, so a gap in the
-- history means nothing was read.
SELECT DISTINCT ON (reading_day)
       reading_day::date AS reading_day, status, storage_percent, storage_fresh,
       flow_kind, flow_cusecs, flow_agency, agency, fetched_at
FROM reservoir_readings
WHERE reservoir_id = $1 AND reading_day IS NOT NULL AND reading_day >= $2
  AND (storage_fresh IS TRUE OR status <> 'nodata')
ORDER BY reading_day, snapshot_day DESC;
