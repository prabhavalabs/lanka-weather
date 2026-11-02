-- name: DeletePastFloodEvents :exec
DELETE FROM past_flood_events;

-- name: InsertPastFloodEvent :exec
-- A heavily fragmented extent (the Puttalam map is 31,985 raster cells) is
-- dissolved here. Districts are those the extent overlaps by more than a
-- hectare, so a boundary brushing past does not count.
WITH shape AS (
    SELECT ST_Multi(ST_CollectionExtract(ST_UnaryUnion(ST_MakeValid(ST_SetSRID(ST_GeomFromWKB(@wkb::bytea), 4326))), 3)) AS g
)
INSERT INTO past_flood_events (fid, id, name, source_name, river, year, event_date, magnitude, area_km2, districts,
                               west, south, east, north, geom, geom_3857, service, source, version, source_time,
                               fetched_at)
SELECT @fid, @id, @name, @source_name, @river, @year, @event_date, @magnitude, ST_Area(g::geography) / 1e6,
       coalesce((SELECT array_agg(d.name ORDER BY d.name)
                 FROM regions d
                 WHERE d.level = 'district' AND ST_Intersects(d.geom, g)
                   AND ST_Area(ST_Intersection(d.geom, g)::geography) > 10000), ARRAY[]::text[]),
       ST_XMin(g), ST_YMin(g), ST_XMax(g), ST_YMax(g), g, ST_Transform(g, 3857), @service, @source, @version,
       @source_time, @fetched_at
FROM shape;

-- name: CountPastFloodEvents :one
SELECT count(*)::int FROM past_flood_events;

-- name: ListPastFloodEvents :many
SELECT fid, id, name, source_name, river, year, event_date, magnitude, area_km2, districts, west, south, east, north,
       service, source, version, source_time, fetched_at
FROM past_flood_events
ORDER BY year DESC, event_date DESC NULLS LAST, name;

-- name: PastFloodTile :one
-- One Mapbox Vector Tile of past flood extents.
WITH bounds AS (
    SELECT ST_TileEnvelope(@z::int, @x::int, @y::int) AS tile,
           ST_TileEnvelope(@z::int, @x::int, @y::int, margin => 64.0 / 4096) AS area
),
events AS (
    SELECT e.fid, e.id, e.name, e.year,
           ST_AsMVTGeom(CASE WHEN @tolerance::float8 > 0 THEN ST_SimplifyPreserveTopology(e.geom_3857, @tolerance::float8) ELSE e.geom_3857 END,
                        bounds.tile, 4096, 64, true) AS geom
    FROM past_flood_events e, bounds
    WHERE e.geom_3857 && bounds.area
)
SELECT coalesce((SELECT ST_AsMVT(f, 'events', 4096, 'geom', 'fid') FROM (SELECT * FROM events WHERE geom IS NOT NULL) f), ''::bytea)::bytea AS tile;
