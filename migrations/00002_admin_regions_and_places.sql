-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- The version of a source that was last loaded, so collectors of rarely
-- changing sources can skip unchanged versions and the API can tag responses
-- with the data version (ETag).
CREATE TABLE source_versions (
    source      text PRIMARY KEY,
    -- Upstream identifier of the version, for example a git commit SHA.
    version     text        NOT NULL,
    -- When the source published this version.
    source_time timestamptz NOT NULL,
    -- When this version was fetched and stored.
    fetched_at  timestamptz NOT NULL,
    records     integer     NOT NULL
);

-- Administrative regions at four levels: province (9), district (25),
-- divisional secretariat division (ds, about 340) and Grama Niladhari
-- division (gn, about 14,000). The id is the Department of Census and
-- Statistics code with a hyphen after the country prefix (LK-1, LK-11,
-- LK-1103, LK-1103005); a region's code starts with its parent's code.
CREATE TABLE regions (
    id               text PRIMARY KEY,
    -- The code's digits as a number (LK-1103005 is 1103005), unique across
    -- levels because each level has a different number of digits. Vector
    -- tiles carry it as the feature id.
    fid              bigint           NOT NULL UNIQUE,
    level            text             NOT NULL CHECK (level IN ('province', 'district', 'ds', 'gn')),
    name             text             NOT NULL,
    name_si          text,
    name_ta          text,
    parent_id        text,
    -- The province, district and DS division containing the region, or the
    -- region itself at its own level; NULL for levels below the region.
    province_id      text             NOT NULL,
    district_id      text,
    ds_id            text,
    -- ISO 3166-2 code of a province or district. It differs from the DCS code
    -- for the Northern Province districts (DCS LK-42 Mannar is ISO LK-43).
    iso_code         text,
    -- A GN division's official number, such as "380" or "520A".
    gn_number        text,
    area_km2         double precision NOT NULL,
    population       integer,
    population_year  smallint,
    -- Area-weighted centre published by the source; it can fall outside a
    -- concave region. label_point is always inside.
    centroid         geometry(Point, 4326)        NOT NULL,
    label_point      geometry(Point, 4326)        NOT NULL,
    west             double precision NOT NULL,
    south            double precision NOT NULL,
    east             double precision NOT NULL,
    north            double precision NOT NULL,
    -- Boundary as published (about 10 m precision), for spatial joins.
    geom             geometry(MultiPolygon, 4326) NOT NULL,
    -- Boundary simplified for the region's level along shared edges, so
    -- neighbours still meet; served as GeoJSON and in low-zoom tiles.
    geom_simple      geometry(MultiPolygon, 4326) NOT NULL,
    geom_3857        geometry(MultiPolygon, 3857) NOT NULL,
    geom_simple_3857 geometry(MultiPolygon, 3857) NOT NULL,
    source           text             NOT NULL,
    source_version   text             NOT NULL,
    source_time      timestamptz      NOT NULL,
    fetched_at       timestamptz      NOT NULL
);
CREATE INDEX regions_level_idx ON regions (level);
CREATE INDEX regions_parent_idx ON regions (parent_id);
CREATE INDEX regions_geom_idx ON regions USING gist (geom);
CREATE INDEX regions_geom_3857_idx ON regions USING gist (geom_3857);

-- A collector loads a complete set of regions here, then merges it into
-- regions in the same transaction.
CREATE UNLOGGED TABLE region_staging (
    id              text             NOT NULL,
    fid             bigint           NOT NULL,
    level           text             NOT NULL,
    name            text             NOT NULL,
    name_si         text,
    name_ta         text,
    parent_id       text,
    province_id     text             NOT NULL,
    district_id     text,
    ds_id           text,
    iso_code        text,
    gn_number       text,
    area_km2        double precision NOT NULL,
    population      integer,
    population_year smallint,
    center_lon      double precision NOT NULL,
    center_lat      double precision NOT NULL,
    -- Well-known binary of the boundary and of its simplified form.
    geom_wkb        bytea            NOT NULL,
    simple_wkb      bytea            NOT NULL
);

-- Populated places: cities, towns, villages, hamlets, suburbs and named
-- localities from OpenStreetMap.
CREATE TABLE places (
    id          text PRIMARY KEY,
    name        text             NOT NULL,
    name_si     text,
    name_ta     text,
    alt_names   text[]           NOT NULL DEFAULT '{}',
    kind        text             NOT NULL,
    population  integer,
    lon         double precision NOT NULL,
    lat         double precision NOT NULL,
    geom        geometry(Point, 4326)
                GENERATED ALWAYS AS (ST_SetSRID(ST_MakePoint(lon, lat), 4326)) STORED,
    -- The GN division containing the place (or the nearest within 3 km for
    -- places on the coast), which gives its hierarchy.
    gn_id       text,
    source      text             NOT NULL,
    source_time timestamptz      NOT NULL,
    fetched_at  timestamptz      NOT NULL
);
CREATE INDEX places_geom_idx ON places USING gist (geom);

CREATE UNLOGGED TABLE place_staging (
    id         text             NOT NULL,
    name       text             NOT NULL,
    name_si    text,
    name_ta    text,
    alt_names  text[]           NOT NULL,
    kind       text             NOT NULL,
    population integer,
    lon        double precision NOT NULL,
    lat        double precision NOT NULL
);

-- Folds text for matching: lower case, accents and joiners removed,
-- punctuation as spaces. Sinhala and Tamil letters are kept as they are.
-- +goose StatementBegin
CREATE FUNCTION search_fold(value text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$
SELECT btrim(regexp_replace(regexp_replace(regexp_replace(
    normalize(lower(value), NFD),
    '[\u0300-\u036f\u200c\u200d]', '', 'g'),
    '[-_.,;:''\u2019"()/&+]', ' ', 'g'),
    '\s+', ' ', 'g'))
$$;
-- +goose StatementEnd

-- Every searchable name of every region and place: one row per name and
-- language, with the target's importance for ranking.
CREATE TABLE search_names (
    target_kind text NOT NULL CHECK (target_kind IN ('region', 'place')),
    target_id   text NOT NULL,
    name        text NOT NULL,
    lang        text NOT NULL,
    folded      text NOT NULL,
    importance  real NOT NULL,
    PRIMARY KEY (target_kind, target_id, folded)
);
CREATE INDEX search_names_folded_trgm_idx ON search_names USING gin (folded gin_trgm_ops);
CREATE INDEX search_names_folded_prefix_idx ON search_names (folded text_pattern_ops);

-- +goose Down
DROP TABLE search_names;
DROP FUNCTION search_fold(text);
DROP TABLE place_staging;
DROP TABLE places;
DROP TABLE region_staging;
DROP TABLE regions;
DROP TABLE source_versions;
DROP EXTENSION IF EXISTS pg_trgm;
