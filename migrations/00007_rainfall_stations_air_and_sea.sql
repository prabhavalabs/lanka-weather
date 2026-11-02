-- +goose Up

-- Department of Meteorology stations. A position comes from a sourced table in
-- code for the stations that report temperature, and from a geocoded list for
-- rain-only stations; `position_source` says which, so the interface can call
-- the second kind approximate.
CREATE TABLE met_stations (
    id              text             PRIMARY KEY,
    name            text             NOT NULL,
    wmo_id          text,
    lon             double precision NOT NULL,
    lat             double precision NOT NULL,
    elevation_m     double precision,
    position_source text             NOT NULL,
    geom            geography(Point, 4326)
                    GENERATED ALWAYS AS (ST_SetSRID(ST_MakePoint(lon, lat), 4326)::geography) STORED,
    source          text             NOT NULL,
    updated_at      timestamptz      NOT NULL
);
CREATE INDEX met_stations_geom_idx ON met_stations USING gist (geom);

-- Three-hourly observations. rain_mm is the rain in the three hours ending at
-- observed_at.
CREATE TABLE met_readings_3h (
    station_id   text             NOT NULL REFERENCES met_stations (id) ON DELETE CASCADE,
    observed_at  timestamptz      NOT NULL,
    rain_mm      double precision,
    temp_c       double precision,
    humidity_pct double precision,
    source       text             NOT NULL,
    fetched_at   timestamptz      NOT NULL,
    PRIMARY KEY (station_id, observed_at)
);
CREATE INDEX met_readings_3h_observed_idx ON met_readings_3h (observed_at);

-- The daily report: rain over the 24 hours ending at 08:30 local time on
-- report_date (observed_at), with the temperature extremes reported with it.
CREATE TABLE met_readings_daily (
    station_id  text             NOT NULL REFERENCES met_stations (id) ON DELETE CASCADE,
    report_date date             NOT NULL,
    observed_at timestamptz      NOT NULL,
    rain_mm     double precision,
    temp_min_c  double precision,
    temp_max_c  double precision,
    source      text             NOT NULL,
    fetched_at  timestamptz      NOT NULL,
    PRIMARY KEY (station_id, report_date)
);
CREATE INDEX met_readings_daily_observed_idx ON met_readings_daily (observed_at);

-- IMERG precipitation tiles over Sri Lanka and nearby seas, collected from
-- NASA GIBS so browsers read them from this API. A NULL png is a tile GIBS
-- rendered with no rain at all. valid_at is the start of the half hour the
-- image covers. PNGs do not compress further, so TOAST does not try.
CREATE TABLE rain_tiles (
    valid_at   timestamptz NOT NULL,
    z          integer     NOT NULL,
    x          integer     NOT NULL,
    y          integer     NOT NULL,
    png        bytea,
    source     text        NOT NULL,
    fetched_at timestamptz NOT NULL,
    PRIMARY KEY (valid_at, z, x, y)
);
ALTER TABLE rain_tiles ALTER COLUMN png SET STORAGE EXTERNAL;

-- Air quality and sea conditions use the weather run tables on grids of their
-- own, so storage, pruning and reading stay the same.
ALTER TABLE weather_runs DROP CONSTRAINT weather_runs_kind_check;
ALTER TABLE weather_runs ADD CONSTRAINT weather_runs_kind_check
    CHECK (kind IN ('current', 'forecast', 'air_quality', 'marine'));

-- +goose Down
DELETE FROM weather_runs WHERE kind IN ('air_quality', 'marine');
ALTER TABLE weather_runs DROP CONSTRAINT weather_runs_kind_check;
ALTER TABLE weather_runs ADD CONSTRAINT weather_runs_kind_check
    CHECK (kind IN ('current', 'forecast'));
DROP TABLE rain_tiles;
DROP TABLE met_readings_daily;
DROP TABLE met_readings_3h;
DROP TABLE met_stations;
