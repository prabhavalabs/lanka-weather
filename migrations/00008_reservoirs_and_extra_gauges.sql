-- +goose Up
-- Reservoirs and tanks as published by Sri Lanka Water Levels (waterlevels.lk),
-- which compiles the Irrigation Department's reservoir sheet and the Mahaweli
-- Authority's daily report. Coordinates are not in that feed: they come from a
-- conservative match to OpenStreetMap, and stay null when nothing matches, so
-- an unmatched reservoir is listed but never guessed on the map.
CREATE TABLE reservoirs (
    id             text PRIMARY KEY,
    name           text        NOT NULL,
    -- The district as the source publishes it; the source gives none for most
    -- Mahaweli Authority reservoirs.
    district       text,
    river          text,
    kind           text        NOT NULL,
    source         text        NOT NULL,
    source_url     text,
    updated_at     timestamptz NOT NULL,
    lon            double precision,
    lat            double precision,
    area           geometry(MultiPolygon, 4326),
    osm_id         text,
    osm_name       text,
    -- 'water' for a mapped water body, 'dam' for a marker at a named dam.
    osm_kind       text CHECK (osm_kind IN ('water', 'dam')),
    map_source     text,
    map_source_time timestamptz,
    map_fetched_at timestamptz,
    CHECK ((lon IS NULL) = (lat IS NULL))
);
CREATE INDEX reservoirs_area_idx ON reservoirs USING gist (area);

-- What the source said about a reservoir on each calendar day (Asia/Colombo),
-- one row a day: later fetches on the same day replace it. reading_day is the
-- day the figures were read, which is older than snapshot_day when an agency
-- has not reported, so a day without a reading stays visible as such and never
-- overwrites the reading it follows.
CREATE TABLE reservoir_readings (
    reservoir_id    text        NOT NULL REFERENCES reservoirs (id) ON DELETE CASCADE,
    snapshot_day    date        NOT NULL,
    reading_day     date,
    status          text        NOT NULL CHECK (status IN ('spilling', 'releasing', 'holding', 'nodata')),
    storage_percent double precision,
    storage_fresh   boolean,
    flow_kind       text,
    flow_cusecs     double precision,
    flow_agency     text,
    flow_day        date,
    agency          text,
    source          text        NOT NULL,
    source_time     timestamptz NOT NULL,
    fetched_at      timestamptz NOT NULL,
    PRIMARY KEY (reservoir_id, snapshot_day)
);

-- Named water bodies and dams whose name may match a reservoir, kept only for
-- the length of the transaction that matches them.
CREATE UNLOGGED TABLE reservoir_candidate_staging (
    osm_id  text NOT NULL PRIMARY KEY,
    lines   text NOT NULL
);

-- Gauges from another collector carry where to read more and who measured them.
ALTER TABLE river_stations
    ADD COLUMN source_url text,
    ADD COLUMN upstream   text;

-- Some gauges publish no unit at all: their level is shown without a verdict.
ALTER TABLE river_stations DROP CONSTRAINT river_stations_unit_check;
ALTER TABLE river_stations ADD CONSTRAINT river_stations_unit_check CHECK (unit IN ('m', 'ft', 'unknown'));

-- Ampara has read exactly 0 at every observation, with no thresholds, and the
-- department publishes no unit for it: the metres label was only a default.
UPDATE river_stations SET unit = 'unknown' WHERE id = 'ampara' AND source = 'irrigation-gauges';

-- The experimental next-hour forecast by Sri Lanka Water Levels. Always in feet
-- as published; it is shown beside the official status and never changes it.
-- Stations are never deleted, and no foreign key means a collector's tests can
-- still truncate the river tables without knowing about this one.
CREATE TABLE river_next_hour (
    station_id   text             NOT NULL,
    issued_at    timestamptz      NOT NULL,
    level_ft     double precision NOT NULL,
    low_ft       double precision,
    high_ft      double precision,
    change_ft    double precision,
    direction    text,
    crossing     text,
    method       text,
    source       text             NOT NULL,
    fetched_at   timestamptz      NOT NULL,
    PRIMARY KEY (station_id, issued_at)
);

-- +goose Down
DROP TABLE river_next_hour;
UPDATE river_stations SET unit = 'm' WHERE unit = 'unknown';
ALTER TABLE river_stations DROP CONSTRAINT river_stations_unit_check;
ALTER TABLE river_stations ADD CONSTRAINT river_stations_unit_check CHECK (unit IN ('m', 'ft'));
ALTER TABLE river_stations DROP COLUMN upstream, DROP COLUMN source_url;
DROP TABLE reservoir_candidate_staging;
DROP TABLE reservoir_readings;
DROP TABLE reservoirs;
