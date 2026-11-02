-- name: DeleteOfficialBasins :exec
DELETE FROM official_basins;

-- name: InsertOfficialBasin :exec
-- Rings are repaired and kept as polygons; label_point is always inside.
WITH shape AS (
    SELECT ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_SetSRID(ST_GeomFromWKB(@wkb::bytea), 4326)), 3)) AS g
)
INSERT INTO official_basins (fid, id, name, named, area_km2, label_point, geom, geom_3857, source, version, source_time,
                             fetched_at)
SELECT @fid, @id, @name, @named, ST_Area(g::geography) / 1e6, (ST_MaximumInscribedCircle(g)).center, g,
       ST_Transform(g, 3857), @source, @version, @source_time, @fetched_at
FROM shape;

-- name: CountOfficialBasins :one
SELECT count(*)::int FROM official_basins;

-- name: DeleteOfficialStreamNames :exec
DELETE FROM official_stream_names;

-- name: InsertOfficialStreamName :exec
INSERT INTO official_stream_names (fid, name, geom, source, fetched_at)
VALUES (@fid, @name,
        ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_SetSRID(ST_GeomFromWKB(@wkb::bytea), 4326)), 3)),
        @source, @fetched_at);

-- name: DeleteOfficialStations :exec
DELETE FROM official_stations;

-- name: InsertOfficialStation :exec
INSERT INTO official_stations (station, basin, tributary, lon, lat, source, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (station) DO NOTHING;

-- name: DeleteRiverSubbasins :exec
DELETE FROM river_subbasins;

-- name: InsertRiverSubbasin :exec
WITH shape AS (
    SELECT ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_SetSRID(ST_GeomFromWKB(@wkb::bytea), 4326)), 3)) AS g
)
INSERT INTO river_subbasins (hybas_id, next_down, main_bas, area_km2, up_area_km2, geom, geom_3857, source, version,
                             source_time, fetched_at)
SELECT @hybas_id, @next_down, @main_bas, @area_km2, @up_area_km2, g, ST_Transform(g, 3857), @source, @version,
       @source_time, @fetched_at
FROM shape;

-- name: CountRiverSubbasins :one
SELECT count(*)::int FROM river_subbasins;

-- name: AssignSubbasinsToOfficialBasins :execrows
-- A sub-basin belongs to the official basin containing most of it.
UPDATE river_subbasins s
SET basin_fid = (
    SELECT b.fid
    FROM official_basins b
    WHERE ST_Intersects(b.geom, s.geom)
    ORDER BY ST_Area(ST_Intersection(b.geom, s.geom)) DESC, b.fid
    LIMIT 1
);

-- name: ListOfficialBasins :many
SELECT b.fid, b.id, b.name, b.named, b.area_km2, b.source_time, ST_X(b.label_point)::float8 AS label_lon,
       ST_Y(b.label_point)::float8 AS label_lat, ST_XMin(b.geom)::float8 AS west, ST_YMin(b.geom)::float8 AS south,
       ST_XMax(b.geom)::float8 AS east, ST_YMax(b.geom)::float8 AS north, b.version, b.fetched_at
FROM official_basins b
ORDER BY b.area_km2 DESC, b.fid;

-- name: ListSubbasins :many
SELECT hybas_id, next_down, main_bas, area_km2, basin_fid
FROM river_subbasins
ORDER BY hybas_id;

-- name: GaugeBasinsAtPoints :many
-- The official basin and sub-basin containing each point; a gauge on the
-- simplified coast takes the nearest within about 3 km.
SELECT p.position::int AS position,
       coalesce((SELECT b.fid FROM official_basins b WHERE ST_Intersects(b.geom, p.pt) ORDER BY b.named DESC, b.area_km2 DESC LIMIT 1),
                (SELECT b.fid FROM official_basins b WHERE ST_DWithin(b.geom, p.pt, 0.03) ORDER BY ST_Distance(b.geom, p.pt) LIMIT 1),
                0)::bigint AS basin_fid,
       coalesce((SELECT s.hybas_id FROM river_subbasins s WHERE ST_Intersects(s.geom, p.pt) LIMIT 1),
                (SELECT s.hybas_id FROM river_subbasins s WHERE ST_DWithin(s.geom, p.pt, 0.03) ORDER BY ST_Distance(s.geom, p.pt) LIMIT 1),
                0)::bigint AS hybas_id
FROM (
    SELECT u.position, ST_SetSRID(ST_MakePoint(u.lon, u.lat), 4326) AS pt
    FROM (SELECT unnest(@positions::int[]) AS position, unnest(@lons::float8[]) AS lon,
                 unnest(@lats::float8[]) AS lat) u
) p
ORDER BY p.position;

-- name: FloodAreasInBasins :many
-- Google's flood areas touching each official basin, within its retention.
SELECT b.fid AS basin_fid, a.id, a.kind, a.severity
FROM flood_areas a
JOIN official_basins b ON ST_Intersects(b.geom, a.geom)
WHERE a.fetched_at >= @since
ORDER BY b.fid, a.id;

-- name: SeedActive :one
-- Whether local test data is present in the live tables.
SELECT (EXISTS (SELECT 1 FROM flood_gauges WHERE source = @source::text)
     OR EXISTS (SELECT 1 FROM flood_areas WHERE source = @source::text)
     OR EXISTS (SELECT 1 FROM river_stations WHERE source = @source::text))::bool AS active;

-- name: BasinTile :one
-- One Mapbox Vector Tile with the official basins and, from a zoom, the
-- sub-basins. Low zooms read simplified shapes: the source's outlines are far
-- finer than a low-zoom tile can show. Coastal basins too small to see at a
-- zoom are left out by size.
WITH bounds AS (
    SELECT ST_TileEnvelope(@z::int, @x::int, @y::int) AS tile,
           ST_TileEnvelope(@z::int, @x::int, @y::int, margin => 64.0 / 4096) AS area
),
basins AS (
    SELECT b.fid, b.id, b.name, b.named, b.area_km2::int AS area,
           ST_AsMVTGeom(CASE WHEN @tolerance::float8 > 0 THEN ST_SimplifyPreserveTopology(b.geom_3857, @tolerance::float8) ELSE b.geom_3857 END,
                        bounds.tile, 4096, 64, true) AS geom
    FROM official_basins b, bounds
    WHERE b.geom_3857 && bounds.area AND b.area_km2 >= @min_area::float8
),
subbasins AS (
    SELECT s.hybas_id AS fid, s.hybas_id::text AS id, coalesce(s.basin_fid, 0)::bigint AS basin,
           ST_AsMVTGeom(s.geom_3857, bounds.tile, 4096, 64, true) AS geom
    FROM river_subbasins s, bounds
    WHERE @with_subbasins::bool AND s.geom_3857 && bounds.area
)
SELECT (coalesce((SELECT ST_AsMVT(f, 'basins', 4096, 'geom', 'fid') FROM (SELECT * FROM basins WHERE geom IS NOT NULL) f), ''::bytea)
     || coalesce((SELECT ST_AsMVT(f, 'subbasins', 4096, 'geom', 'fid') FROM (SELECT * FROM subbasins WHERE geom IS NOT NULL) f), ''::bytea)
       )::bytea AS tile;
