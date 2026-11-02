-- name: ClearRegionStaging :exec
TRUNCATE region_staging;

-- name: StageRegions :copyfrom
INSERT INTO region_staging (id, fid, level, name, name_si, name_ta, parent_id, province_id, district_id, ds_id,
                            iso_code, gn_number, area_km2, population, population_year, center_lon, center_lat,
                            geom_wkb, simple_wkb)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19);

-- name: MergeStagedRegions :execrows
-- Upserts the staged regions. Boundaries are made valid and their derived
-- columns computed here, once per load, so serving never repeats the work.
INSERT INTO regions (id, fid, level, name, name_si, name_ta, parent_id, province_id, district_id, ds_id,
                     iso_code, gn_number, area_km2, population, population_year, centroid, label_point,
                     west, south, east, north, geom, geom_simple, geom_3857, geom_simple_3857,
                     source, source_version, source_time, fetched_at)
SELECT s.id, s.fid, s.level, s.name, s.name_si, s.name_ta, s.parent_id, s.province_id, s.district_id, s.ds_id,
       s.iso_code, s.gn_number,
       CASE WHEN s.area_km2 > 0 THEN s.area_km2 ELSE ST_Area(g.geom::geography) / 1e6 END,
       s.population, s.population_year,
       ST_SetSRID(ST_MakePoint(s.center_lon, s.center_lat), 4326),
       -- The centre of the largest inscribed circle labels a large region
       -- well; a point on the surface is enough for a GN division.
       CASE WHEN s.level = 'gn' THEN ST_PointOnSurface(g.geom)
            ELSE (ST_MaximumInscribedCircle(g.geom)).center END,
       ST_XMin(g.geom), ST_YMin(g.geom), ST_XMax(g.geom), ST_YMax(g.geom),
       g.geom, g.simple, ST_Transform(g.geom, 3857), ST_Transform(g.simple, 3857),
       @source::text, @source_version::text, @source_time::timestamptz, @fetched_at::timestamptz
FROM region_staging s
CROSS JOIN LATERAL (
    SELECT valid.geom,
           CASE WHEN ST_IsEmpty(valid.simple) THEN valid.geom ELSE valid.simple END AS simple
    FROM (SELECT ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_GeomFromWKB(s.geom_wkb, 4326)), 3)) AS geom,
                 ST_Multi(ST_CollectionExtract(ST_MakeValid(ST_GeomFromWKB(s.simple_wkb, 4326)), 3)) AS simple) valid
) g
WHERE NOT ST_IsEmpty(g.geom)
ON CONFLICT (id) DO UPDATE
SET fid = EXCLUDED.fid,
    level = EXCLUDED.level,
    name = EXCLUDED.name,
    name_si = EXCLUDED.name_si,
    name_ta = EXCLUDED.name_ta,
    parent_id = EXCLUDED.parent_id,
    province_id = EXCLUDED.province_id,
    district_id = EXCLUDED.district_id,
    ds_id = EXCLUDED.ds_id,
    iso_code = EXCLUDED.iso_code,
    gn_number = EXCLUDED.gn_number,
    area_km2 = EXCLUDED.area_km2,
    population = EXCLUDED.population,
    population_year = EXCLUDED.population_year,
    centroid = EXCLUDED.centroid,
    label_point = EXCLUDED.label_point,
    west = EXCLUDED.west,
    south = EXCLUDED.south,
    east = EXCLUDED.east,
    north = EXCLUDED.north,
    geom = EXCLUDED.geom,
    geom_simple = EXCLUDED.geom_simple,
    geom_3857 = EXCLUDED.geom_3857,
    geom_simple_3857 = EXCLUDED.geom_simple_3857,
    source = EXCLUDED.source,
    source_version = EXCLUDED.source_version,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: DeleteRegionsNotStaged :execrows
-- Removes regions the latest complete load no longer contains, such as a
-- division that was merged into another.
DELETE FROM regions r
WHERE NOT EXISTS (SELECT 1 FROM region_staging s WHERE s.id = r.id);

-- name: CountRegionsByLevel :many
SELECT level, count(*)::int AS regions
FROM regions
GROUP BY level;

-- name: DeleteSearchNames :exec
DELETE FROM search_names WHERE target_kind = $1;

-- name: InsertRegionSearchNames :execrows
-- Indexes each region's English, Sinhala and Tamil names, and the English
-- name with its level ("Colombo District"). Importance ranks larger and more
-- populous regions first among equally good matches.
INSERT INTO search_names (target_kind, target_id, name, lang, folded, importance)
SELECT 'region', r.id, n.name, n.lang, search_fold(n.name),
       (CASE r.level WHEN 'province' THEN 0.9 WHEN 'district' THEN 0.85 WHEN 'ds' THEN 0.6 ELSE 0.35 END
           + 0.1 * least(1.0, ln(greatest(coalesce(r.population, 1), 1)) / ln(10000000)))::real
FROM regions r
CROSS JOIN LATERAL (
    VALUES (r.name, 'en'),
           (r.name_si, 'si'),
           (r.name_ta, 'ta'),
           (CASE r.level
                WHEN 'province' THEN r.name || ' Province'
                WHEN 'district' THEN r.name || ' District'
                WHEN 'ds' THEN r.name || ' Divisional Secretariat'
            END, 'en')
) AS n(name, lang)
WHERE n.name IS NOT NULL AND search_fold(n.name) <> ''
ON CONFLICT DO NOTHING;

-- name: GetRegion :one
SELECT r.id, r.level, r.name, r.name_si, r.name_ta, r.parent_id, r.province_id, r.district_id, r.ds_id,
       r.iso_code, r.gn_number, r.area_km2, r.population, r.population_year,
       ST_X(r.centroid)::float8 AS centroid_lon, ST_Y(r.centroid)::float8 AS centroid_lat,
       ST_X(r.label_point)::float8 AS label_lon, ST_Y(r.label_point)::float8 AS label_lat,
       r.west, r.south, r.east, r.north, r.source, r.source_version, r.source_time, r.fetched_at,
       (SELECT count(*) FROM regions c WHERE c.parent_id = r.id)::int AS child_count
FROM regions r
WHERE r.id = $1;

-- name: GetRegionGeometry :one
-- The region's simplified boundary as GeoJSON, to six decimals (about 0.1 m).
SELECT ST_AsGeoJSON(geom_simple, 6)::text AS geometry
FROM regions
WHERE id = $1;

-- name: RegionNames :many
SELECT id, level, name, name_si, name_ta
FROM regions
WHERE id = ANY(@ids::text[]);

-- name: ListRegionFeatures :many
-- Regions of one level, optionally within an ancestor, with their simplified
-- boundaries as GeoJSON to five decimals (about 1 m).
SELECT r.id, r.level, r.name, r.name_si, r.name_ta, r.parent_id, r.province_id, r.district_id, r.ds_id,
       r.iso_code, r.gn_number, r.area_km2, r.population, r.population_year,
       ST_X(r.label_point)::float8 AS label_lon, ST_Y(r.label_point)::float8 AS label_lat,
       ST_AsGeoJSON(r.geom_simple, 5)::text AS geometry
FROM regions r
WHERE r.level = @level::text
  AND (@ancestor::text = '' OR @ancestor::text IN (r.parent_id, r.province_id, r.district_id, r.ds_id))
ORDER BY r.id;

-- name: LookupRegions :many
-- The regions containing a point, largest first. A point on a shared border
-- matches both neighbours.
SELECT r.id, r.level, r.name, r.name_si, r.name_ta
FROM regions r
WHERE ST_Intersects(r.geom, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326))
ORDER BY array_position(ARRAY['province', 'district', 'ds', 'gn'], r.level), r.id;

-- name: RegionTile :one
-- One Mapbox Vector Tile with a layer per requested level. Each feature's id
-- is the region's numeric code. simple selects the boundaries simplified per
-- level, for low zooms.
WITH bounds AS (
    SELECT ST_TileEnvelope(@z::int, @x::int, @y::int) AS tile,
           ST_TileEnvelope(@z::int, @x::int, @y::int, margin => 64.0 / 4096) AS area
),
features AS (
    SELECT r.fid, r.id, r.level, r.name, r.name_si AS "nameSi", r.name_ta AS "nameTa",
           r.parent_id AS "parentId", r.province_id AS "provinceId", r.district_id AS "districtId",
           r.ds_id AS "dsId",
           ST_AsMVTGeom(CASE WHEN @simple::bool THEN r.geom_simple_3857 ELSE r.geom_3857 END,
                        bounds.tile, 4096, 64, true) AS geom
    FROM regions r, bounds
    WHERE r.geom_3857 && bounds.area
      AND r.level = ANY(@levels::text[])
)
SELECT (coalesce((SELECT ST_AsMVT(f, 'province', 4096, 'geom', 'fid')
                  FROM (SELECT * FROM features WHERE level = 'province' AND geom IS NOT NULL) f), ''::bytea)
     || coalesce((SELECT ST_AsMVT(f, 'district', 4096, 'geom', 'fid')
                  FROM (SELECT * FROM features WHERE level = 'district' AND geom IS NOT NULL) f), ''::bytea)
     || coalesce((SELECT ST_AsMVT(f, 'ds', 4096, 'geom', 'fid')
                  FROM (SELECT * FROM features WHERE level = 'ds' AND geom IS NOT NULL) f), ''::bytea)
     || coalesce((SELECT ST_AsMVT(f, 'gn', 4096, 'geom', 'fid')
                  FROM (SELECT * FROM features WHERE level = 'gn' AND geom IS NOT NULL) f), ''::bytea)
       )::bytea AS tile;
