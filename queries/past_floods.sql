-- name: DeletePastFloodEvents :exec
DELETE FROM past_flood_events;

-- name: InsertPastFloodEvent :exec
-- The source's extents are traced from rasters, with tens of thousands of rings
-- that touch and overlap. The structure method of ST_MakeValid unions outer
-- rings; on the largest extent it takes 18 s where the default takes over 5 min.
WITH shape AS (
    SELECT ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_SetSRID(ST_GeomFromWKB(@wkb::bytea), 4326), 'method=structure'), 3)) AS g
)
INSERT INTO past_flood_events (fid, id, name, source_name, river, year, event_date, magnitude, area_km2, districts,
                               west, south, east, north, geom, geom_simple_3857, service, source, version, source_time,
                               fetched_at)
SELECT @fid, @id, @name, @source_name, @river, @year, @event_date, @magnitude, ST_Area(g::geography) / 1e6, ARRAY[]::text[],
       ST_XMin(g), ST_YMin(g), ST_XMax(g), ST_YMax(g), g, ST_Multi(ST_Transform(g, 3857)), @service, @source, @version,
       @source_time, @fetched_at
FROM shape;

-- name: BuildPastFloodDetail :exec
-- Cuts an extent into pieces for tiles, simplifies it for low zooms (dropping
-- polygons under half a hectare, too small to see there) and finds the districts
-- it overlaps by more than a hectare, so a boundary brushing past does not count.
WITH parts AS (
    INSERT INTO past_flood_parts (fid, geom_3857)
    SELECT e.fid, ST_Transform(ST_Subdivide(e.geom, 512), 3857)
    FROM past_flood_events e
    WHERE e.fid = @fid
    RETURNING geom_3857
),
pieces AS (
    SELECT ST_Transform(geom_3857, 4326) AS p FROM parts
),
touching AS (
    SELECT d.name
    FROM regions d
    JOIN pieces q ON d.level = 'district' AND d.geom_simple && q.p AND ST_Intersects(d.geom_simple, q.p)
    GROUP BY d.name
    HAVING sum(ST_Area(ST_Intersection(d.geom_simple, q.p)::geography)) > 10000
),
kept AS (
    SELECT ST_Collect(d.geom) AS g
    FROM past_flood_events e, ST_Dump(ST_Transform(e.geom, 3857)) d
    WHERE e.fid = @fid AND ST_Area(d.geom) > 5000
)
UPDATE past_flood_events e
SET districts = coalesce((SELECT array_agg(o.name ORDER BY o.name) FROM touching o), ARRAY[]::text[]),
    geom_simple_3857 = coalesce(
        (SELECT ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_SimplifyPreserveTopology(k.g, 20), 'method=structure'), 3)) FROM kept k),
        e.geom_simple_3857)
WHERE e.fid = @fid;

-- name: CountPastFloodEvents :one
SELECT count(*)::int FROM past_flood_events;

-- name: ListPastFloodEvents :many
SELECT fid, id, name, source_name, river, year, event_date, magnitude, area_km2, districts, west, south, east, north,
       service, source, version, source_time, fetched_at
FROM past_flood_events
ORDER BY year DESC, event_date DESC NULLS LAST, name;

-- name: PastFloodTile :one
-- One Mapbox Vector Tile of past flood extents: simplified shapes at low zooms,
-- the full ones joined back from their pieces close up.
WITH bounds AS (
    SELECT ST_TileEnvelope(@z::int, @x::int, @y::int) AS tile,
           ST_TileEnvelope(@z::int, @x::int, @y::int, margin => 64.0 / 4096) AS area
),
near AS (
    SELECT p.fid, ST_Union(p.geom_3857) AS g
    FROM past_flood_parts p, bounds
    WHERE @use_parts::bool AND p.geom_3857 && bounds.area
    GROUP BY p.fid
),
far AS (
    SELECT e.fid,
           CASE WHEN @tolerance::float8 > 0 THEN ST_SimplifyPreserveTopology(e.geom_simple_3857, @tolerance::float8) ELSE e.geom_simple_3857 END AS g
    FROM past_flood_events e, bounds
    WHERE NOT @use_parts::bool AND e.geom_simple_3857 && bounds.area
),
events AS (
    SELECT e.fid, e.id, e.name, e.year,
           ST_AsMVTGeom(q.g, bounds.tile, 4096, 64, true) AS geom
    FROM (SELECT * FROM near UNION ALL SELECT * FROM far) q
    JOIN past_flood_events e ON e.fid = q.fid, bounds
)
SELECT coalesce((SELECT ST_AsMVT(f, 'events', 4096, 'geom', 'fid') FROM (SELECT * FROM events WHERE geom IS NOT NULL) f), ''::bytea)::bytea AS tile;
