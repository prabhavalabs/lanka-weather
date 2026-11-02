-- name: ClearRiverSegmentStaging :exec
TRUNCATE river_segment_staging;

-- name: StageRiverSegments :copyfrom
INSERT INTO river_segment_staging (hyriv_id, next_down, main_riv, length_km, dist_dn_km, dist_up_km, catch_km2,
                                   upland_km2, discharge_cms, ord_stra, ord_clas, ord_flow, hybas_l12, wkb)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: MergeStagedRiverSegments :execrows
INSERT INTO river_segments (hyriv_id, next_down, main_riv, length_km, dist_dn_km, dist_up_km, catch_km2, upland_km2,
                            discharge_cms, ord_stra, ord_clas, ord_flow, hybas_l12, geom, geom_3857, source,
                            source_version, source_time, fetched_at)
SELECT s.hyriv_id, s.next_down, s.main_riv, s.length_km, s.dist_dn_km, s.dist_up_km, s.catch_km2, s.upland_km2,
       s.discharge_cms, s.ord_stra, s.ord_clas, s.ord_flow, s.hybas_l12,
       ST_SetSRID(ST_GeomFromWKB(s.wkb), 4326), ST_Transform(ST_SetSRID(ST_GeomFromWKB(s.wkb), 4326), 3857),
       @source::text, @source_version::text, @source_time::timestamptz, @fetched_at::timestamptz
FROM river_segment_staging s
ON CONFLICT (hyriv_id) DO UPDATE
SET next_down = EXCLUDED.next_down,
    main_riv = EXCLUDED.main_riv,
    length_km = EXCLUDED.length_km,
    dist_dn_km = EXCLUDED.dist_dn_km,
    dist_up_km = EXCLUDED.dist_up_km,
    catch_km2 = EXCLUDED.catch_km2,
    upland_km2 = EXCLUDED.upland_km2,
    discharge_cms = EXCLUDED.discharge_cms,
    ord_stra = EXCLUDED.ord_stra,
    ord_clas = EXCLUDED.ord_clas,
    ord_flow = EXCLUDED.ord_flow,
    hybas_l12 = EXCLUDED.hybas_l12,
    geom = EXCLUDED.geom,
    geom_3857 = EXCLUDED.geom_3857,
    source = EXCLUDED.source,
    source_version = EXCLUDED.source_version,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: DeleteRiverSegmentsNotStaged :execrows
DELETE FROM river_segments r
WHERE NOT EXISTS (SELECT 1 FROM river_segment_staging s WHERE s.hyriv_id = r.hyriv_id);

-- name: DeleteRiverSourceNames :exec
DELETE FROM river_source_names;

-- name: InsertRiverSourceName :exec
INSERT INTO river_source_names (riv_id, name, source, fetched_at) VALUES ($1, $2, $3, $4);

-- name: CountRiverSegments :one
SELECT count(*)::int FROM river_segments;

-- name: ListRiverSourceNames :many
SELECT riv_id, name FROM river_source_names ORDER BY riv_id;

-- name: ListSegmentTopology :many
-- Everything the network build and the rain accumulation need, without the
-- lines. The midpoint places a segment's local catchment on the weather grid.
SELECT hyriv_id, next_down, main_riv, length_km, catch_km2, upland_km2, hybas_l12,
       ST_X(ST_LineInterpolatePoint(geom, 0.5))::float8 AS mid_lon,
       ST_Y(ST_LineInterpolatePoint(geom, 0.5))::float8 AS mid_lat
FROM river_segments
ORDER BY upland_km2, hyriv_id;

-- name: SegmentsNearPoint :many
-- Segments within a distance of a point, nearest first.
SELECT hyriv_id, main_riv, upland_km2, hybas_l12,
       ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326)::geography)::float8 AS distance_m
FROM river_segments
WHERE ST_DWithin(geom, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326), @degrees::float8)
ORDER BY distance_m;

-- name: SystemBasinVotes :many
-- How much of each river system's local catchment lies in each named official
-- basin, by the segment midpoints.
SELECT s.main_riv, b.fid AS basin_fid, b.name AS basin_name, sum(s.catch_km2)::float8 AS catch_km2
FROM river_segments s
JOIN official_basins b ON b.named AND ST_Intersects(b.geom, ST_LineInterpolatePoint(s.geom, 0.5))
GROUP BY s.main_riv, b.fid, b.name;

-- name: StreamNameVotes :many
-- Segments running along a named stream polygon of the department's map. A
-- segment counts for a name when most of it lies within 200 m of that stream.
SELECT s.hyriv_id, n.name, count(*)::int AS polygons
FROM river_segments s
JOIN official_stream_names n ON ST_DWithin(s.geom, n.geom, 0.002)
WHERE s.upland_km2 >= @min_upland::float8
  AND ST_DWithin(ST_LineInterpolatePoint(s.geom, 0.5), n.geom, 0.002)
GROUP BY s.hyriv_id, n.name;

-- name: ListOfficialStations :many
SELECT station, basin, tributary, lon, lat FROM official_stations ORDER BY station;

-- name: ListNetworkStations :many
-- Every station with its source: the irrigation collector is not the only
-- one that adds stations.
SELECT id, name, basin, lon, lat, source FROM river_stations ORDER BY id;

-- name: ListNetworkFloodGauges :many
SELECT gauge_id, lon, lat, quality_verified FROM flood_gauges WHERE fetched_at >= @since ORDER BY gauge_id;

-- name: DeleteRivers :exec
DELETE FROM rivers;

-- name: DeleteRiverGaugeMatches :exec
DELETE FROM river_gauge_matches;

-- name: DeleteGoogleGaugeMatchesBefore :exec
DELETE FROM river_gauge_matches WHERE kind = 'google' AND matched_at < $1;

-- name: TouchGoogleGaugeMatches :exec
UPDATE river_gauge_matches SET matched_at = $1 WHERE kind = 'google';

-- name: ClearSegmentAssignments :exec
UPDATE river_segments SET river_id = NULL, reach_fid = NULL
WHERE river_id IS NOT NULL OR reach_fid IS NOT NULL;

-- name: InsertRiver :exec
INSERT INTO rivers (id, name, name_si, aliases, named, parent_id, main_riv, mouth_id, length_km, upland_km2, basin_fid,
                    built_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: InsertRiverReach :exec
INSERT INTO river_reaches (fid, river_id, name, gauged, length_km, upland_km2, position)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: InsertRiverReachGauge :exec
INSERT INTO river_reach_gauges (reach_fid, kind, gauge_id, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING;

-- name: InsertRiverGaugeMatch :exec
INSERT INTO river_gauge_matches (kind, gauge_id, hyriv_id, river_id, distance_m, method, matched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: AssignSegments :exec
UPDATE river_segments s
SET river_id = NULLIF(a.river_id, ''),
    reach_fid = NULLIF(a.reach_fid, 0)
FROM (SELECT unnest(@ids::bigint[]) AS id, unnest(@river_ids::text[]) AS river_id,
             unnest(@reach_fids::bigint[]) AS reach_fid) a
WHERE s.hyriv_id = a.id;

-- name: BuildReachGeometry :execrows
-- Merges each reach's segments into one line for spatial queries.
UPDATE river_reaches r
SET geom = q.geom
FROM (
    SELECT reach_fid, ST_Multi(ST_LineMerge(ST_Collect(geom))) AS geom
    FROM river_segments
    WHERE reach_fid IS NOT NULL
    GROUP BY reach_fid
) q
WHERE q.reach_fid = r.fid;

-- name: BuildReachParts :execrows
-- The lines drawn on the map: per reach and width class, smoothed a little
-- because HydroRIVERS lines follow a 15 arc-second grid. Named rivers are
-- always drawn; unnamed ones from zoom 7 or 8, by size.
INSERT INTO river_reach_parts (reach_fid, width_class, river_id, river_name, named, min_zoom, geom_3857)
SELECT q.reach_fid, q.width_class, rv.id, rv.name, rv.named,
       CASE WHEN rv.named THEN 0 WHEN rv.upland_km2 >= 300 THEN 7 ELSE 8 END,
       ST_Multi(ST_ChaikinSmoothing(q.geom, 1))
FROM (
    SELECT s.reach_fid,
           (CASE WHEN s.upland_km2 >= 3000 THEN 5
                 WHEN s.upland_km2 >= 1000 THEN 4
                 WHEN s.upland_km2 >= 300 THEN 3
                 WHEN s.upland_km2 >= 100 THEN 2
                 ELSE 1 END)::smallint AS width_class,
           ST_LineMerge(ST_Collect(s.geom_3857)) AS geom,
           min(s.river_id) AS river_id
    FROM river_segments s
    WHERE s.reach_fid IS NOT NULL
    GROUP BY s.reach_fid, 2
) q
JOIN river_reaches r ON r.fid = q.reach_fid
JOIN rivers rv ON rv.id = r.river_id;

-- name: BuildRiverExtents :execrows
-- Each river's box, and the merged line its name is placed along.
UPDATE rivers rv
SET west = ST_XMin(q.box), south = ST_YMin(q.box), east = ST_XMax(q.box), north = ST_YMax(q.box),
    label_geom = q.line
FROM (
    SELECT s.river_id, ST_Extent(s.geom)::box2d AS box,
           ST_Multi(ST_LineMerge(ST_Collect(s.geom_3857))) AS line
    FROM river_segments s
    WHERE s.river_id IS NOT NULL AND s.reach_fid IS NOT NULL
    GROUP BY s.river_id
) q
WHERE q.river_id = rv.id;

-- name: NameUnnamedRivers :execrows
-- Unnamed rivers are told apart by the district of their mouth.
UPDATE rivers rv
SET name = 'Unnamed river near ' || d.name
FROM river_segments m
JOIN regions d ON d.level = 'district' AND ST_DWithin(d.geom, ST_EndPoint(m.geom), 0.05)
WHERE NOT rv.named AND m.hyriv_id = rv.mouth_id;

-- name: RiverDrawnSegments :many
SELECT count(*) FILTER (WHERE reach_fid IS NOT NULL)::int AS reached,
       count(*) FILTER (WHERE river_id IS NOT NULL)::int AS assigned,
       count(*)::int AS total
FROM river_segments;

-- name: ListRivers :many
SELECT id, name, name_si, aliases, named, parent_id, main_riv, mouth_id, length_km, upland_km2, basin_fid,
       west, south, east, north, built_at
FROM rivers
ORDER BY upland_km2 DESC, id;

-- name: ListRiverReaches :many
SELECT fid, river_id, name, gauged, length_km, upland_km2, position
FROM river_reaches
ORDER BY river_id, position, fid;

-- name: ListRiverReachGauges :many
SELECT reach_fid, kind, gauge_id, role
FROM river_reach_gauges
ORDER BY reach_fid, role, gauge_id;

-- name: ListRiverGaugeMatches :many
SELECT kind, gauge_id, hyriv_id, river_id, distance_m, method, matched_at
FROM river_gauge_matches
ORDER BY kind, gauge_id;

-- name: ReachRegions :many
-- The districts and DS divisions each stretch passes through.
SELECT q.fid::bigint AS reach_fid, r.id, r.level, r.name, r.name_si
FROM river_reaches q
JOIN regions r ON r.level IN ('district', 'ds') AND ST_Intersects(r.geom, q.geom)
WHERE q.fid = ANY(@fids::bigint[])
ORDER BY q.fid, r.level, r.name;

-- name: RiversNear :many
-- The stretches within a distance of a point, nearest first, one per river.
SELECT DISTINCT ON (q.river_id) q.river_id, q.fid,
       ST_Distance(q.geom::geography, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326)::geography)::float8 AS distance_m
FROM river_reaches q
WHERE ST_DWithin(q.geom, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326), @degrees::float8)
ORDER BY q.river_id, distance_m;

-- name: RiverTile :one
-- One Mapbox Vector Tile with the drawn reaches, the other streams by zoom and
-- the lines names are placed along.
WITH bounds AS (
    SELECT ST_TileEnvelope(@z::int, @x::int, @y::int) AS tile,
           ST_TileEnvelope(@z::int, @x::int, @y::int, margin => 64.0 / 4096) AS area
),
reaches AS (
    SELECT p.reach_fid AS fid, p.reach_fid AS reach, p.river_id AS river, p.river_name AS name, p.named,
           p.width_class AS width,
           ST_AsMVTGeom(p.geom_3857, bounds.tile, 4096, 64, true) AS geom
    FROM river_reach_parts p, bounds
    WHERE p.geom_3857 && bounds.area AND p.min_zoom <= @z::int
),
streams AS (
    SELECT s.hyriv_id AS fid, coalesce(s.river_id, '') AS river,
           (CASE WHEN s.upland_km2 >= 100 THEN 2 ELSE 1 END)::int AS width,
           ST_AsMVTGeom(s.geom_3857, bounds.tile, 4096, 64, true) AS geom
    FROM river_segments s, bounds
    WHERE s.geom_3857 && bounds.area AND s.reach_fid IS NULL AND s.upland_km2 >= @min_upland::float8
),
labels AS (
    SELECT rv.id AS river, rv.name, (rv.parent_id IS NULL)::boolean AS main, rv.upland_km2::int AS upland,
           ST_AsMVTGeom(rv.label_geom, bounds.tile, 4096, 64, true) AS geom
    FROM rivers rv, bounds
    WHERE rv.named AND rv.label_geom && bounds.area
)
SELECT (coalesce((SELECT ST_AsMVT(f, 'reaches', 4096, 'geom', 'fid') FROM (SELECT * FROM reaches WHERE geom IS NOT NULL) f), ''::bytea)
     || coalesce((SELECT ST_AsMVT(f, 'streams', 4096, 'geom', 'fid') FROM (SELECT * FROM streams WHERE geom IS NOT NULL) f), ''::bytea)
     || coalesce((SELECT ST_AsMVT(f, 'labels', 4096, 'geom') FROM (SELECT * FROM labels WHERE geom IS NOT NULL) f), ''::bytea)
       )::bytea AS tile;
