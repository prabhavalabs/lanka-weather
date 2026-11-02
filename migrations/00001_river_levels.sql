-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

-- One row per fetch attempt of a data source, successful or not. The latest
-- successful run tells the API and the web app how fresh a layer is.
CREATE TABLE source_runs (
    id          bigserial PRIMARY KEY,
    source      text        NOT NULL,
    started_at  timestamptz NOT NULL,
    finished_at timestamptz,
    status      text        NOT NULL CHECK (status IN ('running', 'ok', 'error')),
    records     integer     NOT NULL DEFAULT 0,
    error       text
);
CREATE INDEX source_runs_source_started_idx ON source_runs (source, started_at DESC);

-- River gauging stations. Thresholds are in the station's own unit, the same
-- unit as its readings.
CREATE TABLE river_stations (
    id                text PRIMARY KEY,
    name              text             NOT NULL,
    basin             text             NOT NULL,
    lon               double precision NOT NULL,
    lat               double precision NOT NULL,
    geom              geography(Point, 4326)
                      GENERATED ALWAYS AS (ST_SetSRID(ST_MakePoint(lon, lat), 4326)::geography) STORED,
    unit              text             NOT NULL CHECK (unit IN ('m', 'ft')),
    alert_level       double precision,
    minor_flood_level double precision,
    major_flood_level double precision,
    source            text             NOT NULL,
    updated_at        timestamptz      NOT NULL
);
CREATE INDEX river_stations_geom_idx ON river_stations USING gist (geom);

-- Water-level readings. A reading is identified by its station and the time
-- the source observed it, so re-fetching overlapping windows is harmless.
CREATE TABLE river_readings (
    station_id  text             NOT NULL REFERENCES river_stations (id) ON DELETE CASCADE,
    observed_at timestamptz      NOT NULL,
    water_level double precision NOT NULL,
    rainfall_mm double precision,
    fetched_at  timestamptz      NOT NULL,
    PRIMARY KEY (station_id, observed_at)
);
CREATE INDEX river_readings_observed_idx ON river_readings (observed_at DESC);

-- +goose Down
DROP TABLE river_readings;
DROP TABLE river_stations;
DROP TABLE source_runs;
