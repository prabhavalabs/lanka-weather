-- +goose Up

-- HydroRIVERS line segments of Sri Lanka, as published by nuuuwan/lk_rivers.
-- next_down is the segment the water flows into, 0 at the sea. river_id and
-- reach_fid are assigned by the network build (internal/rivernet).
CREATE TABLE river_segments (
    hyriv_id       bigint PRIMARY KEY,
    next_down      bigint           NOT NULL,
    main_riv       bigint           NOT NULL,
    length_km      double precision NOT NULL,
    dist_dn_km     double precision NOT NULL,
    dist_up_km     double precision NOT NULL,
    catch_km2      double precision NOT NULL,
    upland_km2     double precision NOT NULL,
    discharge_cms  double precision NOT NULL,
    ord_stra       smallint         NOT NULL,
    ord_clas       smallint         NOT NULL,
    ord_flow       smallint         NOT NULL,
    hybas_l12      bigint           NOT NULL,
    geom           geometry(LineString, 4326) NOT NULL,
    geom_3857      geometry(LineString, 3857) NOT NULL,
    river_id       text,
    reach_fid      bigint,
    source         text             NOT NULL,
    source_version text             NOT NULL,
    source_time    timestamptz      NOT NULL,
    fetched_at     timestamptz      NOT NULL
);
CREATE INDEX river_segments_geom_idx ON river_segments USING gist (geom);
CREATE INDEX river_segments_geom_3857_idx ON river_segments USING gist (geom_3857);
CREATE INDEX river_segments_main_idx ON river_segments (main_riv);
CREATE INDEX river_segments_hybas_idx ON river_segments (hybas_l12);

CREATE UNLOGGED TABLE river_segment_staging (
    hyriv_id      bigint           NOT NULL,
    next_down     bigint           NOT NULL,
    main_riv      bigint           NOT NULL,
    length_km     double precision NOT NULL,
    dist_dn_km    double precision NOT NULL,
    dist_up_km    double precision NOT NULL,
    catch_km2     double precision NOT NULL,
    upland_km2    double precision NOT NULL,
    discharge_cms double precision NOT NULL,
    ord_stra      smallint         NOT NULL,
    ord_clas      smallint         NOT NULL,
    ord_flow      smallint         NOT NULL,
    hybas_l12     bigint           NOT NULL,
    wkb           bytea            NOT NULL
);

-- Names nuuuwan/lk_rivers gives HydroRIVERS ids: an outlet names a whole
-- river, any other segment is the source of a tributary.
CREATE TABLE river_source_names (
    riv_id     bigint PRIMARY KEY,
    name       text        NOT NULL,
    source     text        NOT NULL,
    fetched_at timestamptz NOT NULL
);

-- The Irrigation Department's river basins (river_basins service): the
-- country-level boundaries of the flood layer. fid is the service's feature
-- id, used as the vector tile feature id.
CREATE TABLE official_basins (
    fid         bigint PRIMARY KEY,
    id          text             NOT NULL UNIQUE,
    name        text             NOT NULL,
    named       boolean          NOT NULL,
    area_km2    double precision NOT NULL,
    label_point geometry(Point, 4326)        NOT NULL,
    geom        geometry(MultiPolygon, 4326) NOT NULL,
    geom_3857   geometry(MultiPolygon, 3857) NOT NULL,
    source      text             NOT NULL,
    version     text             NOT NULL,
    source_time timestamptz      NOT NULL,
    fetched_at  timestamptz      NOT NULL
);
CREATE INDEX official_basins_geom_idx ON official_basins USING gist (geom);
CREATE INDEX official_basins_geom_3857_idx ON official_basins USING gist (geom_3857);

-- Named stream polygons of the department's Srilanka_River service, used only
-- to name HydroRIVERS tributaries.
CREATE TABLE official_stream_names (
    fid        bigint PRIMARY KEY,
    name       text        NOT NULL,
    geom       geometry(MultiPolygon, 4326) NOT NULL,
    source     text        NOT NULL,
    fetched_at timestamptz NOT NULL
);
CREATE INDEX official_stream_names_geom_idx ON official_stream_names USING gist (geom);

-- The department's station list (hydrostations service), used for the
-- tributary each station stands on.
CREATE TABLE official_stations (
    station    text PRIMARY KEY,
    basin      text             NOT NULL,
    tributary  text             NOT NULL,
    lon        double precision NOT NULL,
    lat        double precision NOT NULL,
    source     text             NOT NULL,
    fetched_at timestamptz      NOT NULL
);

-- HydroBASINS level 12 sub-basins of Sri Lanka: the finer boundaries shown
-- when zoomed in. basin_fid is the official basin holding most of it.
CREATE TABLE river_subbasins (
    hybas_id    bigint PRIMARY KEY,
    next_down   bigint           NOT NULL,
    main_bas    bigint           NOT NULL,
    area_km2    double precision NOT NULL,
    up_area_km2 double precision NOT NULL,
    basin_fid   bigint,
    geom        geometry(MultiPolygon, 4326) NOT NULL,
    geom_3857   geometry(MultiPolygon, 3857) NOT NULL,
    source      text             NOT NULL,
    version     text             NOT NULL,
    source_time timestamptz      NOT NULL,
    fetched_at  timestamptz      NOT NULL
);
CREATE INDEX river_subbasins_geom_idx ON river_subbasins USING gist (geom);
CREATE INDEX river_subbasins_geom_3857_idx ON river_subbasins USING gist (geom_3857);

-- The derived network. Everything below is replaced whenever the network is
-- rebuilt, and can be rebuilt at any time from the tables above and the
-- gauges.
CREATE TABLE rivers (
    id           text PRIMARY KEY,
    name         text             NOT NULL,
    name_si      text,
    aliases      text[]           NOT NULL,
    named        boolean          NOT NULL,
    parent_id    text,
    main_riv     bigint           NOT NULL,
    mouth_id     bigint           NOT NULL,
    length_km    double precision NOT NULL,
    upland_km2   double precision NOT NULL,
    basin_fid    bigint,
    west         double precision NOT NULL DEFAULT 0,
    south        double precision NOT NULL DEFAULT 0,
    east         double precision NOT NULL DEFAULT 0,
    north        double precision NOT NULL DEFAULT 0,
    label_geom   geometry(MultiLineString, 3857),
    built_at     timestamptz      NOT NULL
);
CREATE INDEX rivers_label_geom_idx ON rivers USING gist (label_geom);

-- A stretch of river governed by the same gauges. fid is the HydroRIVERS id
-- of its most downstream segment, so it survives a rebuild.
CREATE TABLE river_reaches (
    fid        bigint PRIMARY KEY,
    river_id   text             NOT NULL REFERENCES rivers (id) ON DELETE CASCADE,
    name       text             NOT NULL,
    gauged     boolean          NOT NULL,
    length_km  double precision NOT NULL,
    upland_km2 double precision NOT NULL,
    position   integer          NOT NULL,
    geom       geometry(MultiLineString, 4326)
);
CREATE INDEX river_reaches_geom_idx ON river_reaches USING gist (geom);
CREATE INDEX river_reaches_river_idx ON river_reaches (river_id);

-- Gauges governing a reach. role is upstream (water passing the gauge reaches
-- the stretch), downstream (the next gauge below it) or next (the first gauge
-- below an ungauged stretch, named but not used as evidence).
CREATE TABLE river_reach_gauges (
    reach_fid bigint NOT NULL REFERENCES river_reaches (fid) ON DELETE CASCADE,
    kind      text   NOT NULL CHECK (kind IN ('station', 'google')),
    gauge_id  text   NOT NULL,
    role      text   NOT NULL CHECK (role IN ('upstream', 'downstream', 'next')),
    PRIMARY KEY (reach_fid, kind, gauge_id, role)
);

-- Lines drawn on the map: a reach's segments merged per width class, so the
-- line widens downstream and a reach can still be recoloured as one.
CREATE TABLE river_reach_parts (
    reach_fid   bigint   NOT NULL REFERENCES river_reaches (fid) ON DELETE CASCADE,
    width_class smallint NOT NULL,
    river_id    text     NOT NULL,
    river_name  text     NOT NULL,
    named       boolean  NOT NULL,
    min_zoom    smallint NOT NULL,
    geom_3857   geometry(MultiLineString, 3857) NOT NULL,
    PRIMARY KEY (reach_fid, width_class)
);
CREATE INDEX river_reach_parts_geom_idx ON river_reach_parts USING gist (geom_3857);

-- Where each gauge meets the network. Google's rows are kept no longer than
-- its terms allow: they are replaced hourly and pruned after a day.
CREATE TABLE river_gauge_matches (
    kind       text             NOT NULL CHECK (kind IN ('station', 'google')),
    gauge_id   text             NOT NULL,
    hyriv_id   bigint           NOT NULL,
    river_id   text,
    distance_m double precision NOT NULL,
    method     text             NOT NULL,
    matched_at timestamptz      NOT NULL,
    PRIMARY KEY (kind, gauge_id)
);

-- Mapped extents of past floods from the Irrigation Department (Flood_Map,
-- Flood__Nov2025, Puttalam_Flood__Detwah_30Nov2025). They record events; they
-- are not forecasts.
CREATE TABLE past_flood_events (
    fid            bigint PRIMARY KEY,
    id             text             NOT NULL UNIQUE,
    name           text             NOT NULL,
    source_name    text             NOT NULL,
    river          text             NOT NULL,
    year           integer          NOT NULL,
    event_date     date,
    magnitude      text,
    area_km2       double precision NOT NULL,
    districts      text[]           NOT NULL,
    west           double precision NOT NULL,
    south          double precision NOT NULL,
    east           double precision NOT NULL,
    north          double precision NOT NULL,
    geom           geometry(MultiPolygon, 4326) NOT NULL,
    geom_simple_3857 geometry(MultiPolygon, 3857) NOT NULL,
    service        text             NOT NULL,
    source         text             NOT NULL,
    version        text             NOT NULL,
    source_time    timestamptz      NOT NULL,
    fetched_at     timestamptz      NOT NULL
);
CREATE INDEX past_flood_events_simple_idx ON past_flood_events USING gist (geom_simple_3857);

-- Extents are traced from rasters: one holds 55,000 polygons and 400,000
-- vertices. Close-up tiles read them cut into small pieces, so a tile touches
-- only what lies inside it.
CREATE TABLE past_flood_parts (
    fid       bigint NOT NULL REFERENCES past_flood_events (fid) ON DELETE CASCADE,
    geom_3857 geometry(Polygon, 3857) NOT NULL
);
CREATE INDEX past_flood_parts_geom_idx ON past_flood_parts USING gist (geom_3857);

-- +goose Down
DROP TABLE past_flood_parts;
DROP TABLE past_flood_events;
DROP TABLE river_gauge_matches;
DROP TABLE river_reach_parts;
DROP TABLE river_reach_gauges;
DROP TABLE river_reaches;
DROP TABLE rivers;
DROP TABLE river_subbasins;
DROP TABLE official_stations;
DROP TABLE official_stream_names;
DROP TABLE official_basins;
DROP TABLE river_source_names;
DROP TABLE river_segment_staging;
DROP TABLE river_segments;
