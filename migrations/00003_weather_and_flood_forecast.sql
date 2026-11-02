-- +goose Up

-- Nodes of the weather grid (internal/grid). The grid itself is defined in
-- code; its nodes are stored so regions can be joined to them in PostGIS.
-- A node's cell is the square of one grid step centred on it.
CREATE TABLE weather_grid_nodes (
    grid        text             NOT NULL,
    node        integer          NOT NULL,
    lon         double precision NOT NULL,
    lat         double precision NOT NULL,
    land        boolean          NOT NULL,
    elevation_m double precision,
    cell        geometry(Polygon, 4326) NOT NULL,
    geom        geography(Point, 4326)
                GENERATED ALWAYS AS (ST_SetSRID(ST_MakePoint(lon, lat), 4326)::geography) STORED,
    source      text             NOT NULL,
    updated_at  timestamptz      NOT NULL,
    PRIMARY KEY (grid, node)
);
CREATE INDEX weather_grid_nodes_cell_idx ON weather_grid_nodes USING gist (cell);

-- One fetch of the whole grid: current conditions, or the hourly forecast
-- with the daily summary derived from it. `nodes` lists the lattice index of
-- each value position in the run's fields.
CREATE TABLE weather_runs (
    id          bigserial PRIMARY KEY,
    kind        text        NOT NULL CHECK (kind IN ('current', 'forecast')),
    source      text        NOT NULL,
    model       text        NOT NULL,
    grid        text        NOT NULL,
    nodes       integer[]   NOT NULL,
    -- current: the time the source's values are valid; forecast: first hour.
    source_time timestamptz NOT NULL,
    fetched_at  timestamptz NOT NULL
);
CREATE INDEX weather_runs_kind_idx ON weather_runs (kind, id DESC);

-- One variable of a run over every node and step: steps × nodes values,
-- step-major, in the run's node order. NaN marks a value the source did not
-- give. Steps start at valid_from and are step_seconds apart (0 for a
-- single current value).
CREATE TABLE weather_fields (
    run_id       bigint      NOT NULL REFERENCES weather_runs (id) ON DELETE CASCADE,
    variable     text        NOT NULL,
    source       text        NOT NULL,
    valid_from   timestamptz NOT NULL,
    step_seconds integer     NOT NULL CHECK (step_seconds >= 0),
    steps        integer     NOT NULL CHECK (steps > 0),
    node_values  real[]      NOT NULL,
    fetched_at   timestamptz NOT NULL,
    PRIMARY KEY (run_id, variable)
);

-- Wind 10 m above ground from NOAA GFS, cropped to the seas around Sri
-- Lanka. U (eastward) and V (northward) are little-endian int16s of `scale`
-- m/s in grid order: value j × nx + i lies at lat0 + j × dlat, lon0 + i × dlon.
CREATE TABLE wind_fields (
    run_at        timestamptz      NOT NULL,
    forecast_hour integer          NOT NULL CHECK (forecast_hour >= 0),
    valid_at      timestamptz      NOT NULL,
    source        text             NOT NULL,
    model         text             NOT NULL,
    level         text             NOT NULL,
    lon0          double precision NOT NULL,
    lat0          double precision NOT NULL,
    dlon          double precision NOT NULL,
    dlat          double precision NOT NULL,
    nx            integer          NOT NULL,
    ny            integer          NOT NULL,
    scale         double precision NOT NULL,
    u             bytea            NOT NULL,
    v             bytea            NOT NULL,
    fetched_at    timestamptz      NOT NULL,
    PRIMARY KEY (run_at, forecast_hour)
);
CREATE INDEX wind_fields_valid_idx ON wind_fields (valid_at, run_at DESC);

-- Latest Google Flood Forecasting status of each gauge in Sri Lanka.
-- Google's terms allow gauge search results to be kept for at most a day;
-- every run replaces the set and rows older than a day are deleted.
CREATE TABLE flood_gauges (
    gauge_id             text             PRIMARY KEY,
    source               text             NOT NULL,
    gauge_source         text             NOT NULL,
    site_name            text             NOT NULL,
    river                text             NOT NULL,
    lon                  double precision NOT NULL,
    lat                  double precision NOT NULL,
    geom                 geography(Point, 4326)
                         GENERATED ALWAYS AS (ST_SetSRID(ST_MakePoint(lon, lat), 4326)::geography) STORED,
    quality_verified     boolean          NOT NULL,
    severity             text             NOT NULL,
    trend                text             NOT NULL,
    issued_at            timestamptz      NOT NULL,
    forecast_start       timestamptz,
    forecast_end         timestamptz,
    warning_level        double precision,
    danger_level         double precision,
    extreme_danger_level double precision,
    value_unit           text             NOT NULL,
    fetched_at           timestamptz      NOT NULL
);
CREATE INDEX flood_gauges_geom_idx ON flood_gauges USING gist (geom);

-- Flood areas Google publishes for Sri Lanka: a gauge's notification and
-- inundation polygons, flash-flood areas and significant-event areas. Kept
-- for at most a day, like the gauge statuses they come with.
CREATE TABLE flood_areas (
    id          text        PRIMARY KEY,
    kind        text        NOT NULL CHECK (kind IN ('notification', 'inundation_high', 'inundation_medium',
                                                     'inundation_low', 'flash_likely', 'flash_highly_likely',
                                                     'significant_event')),
    source      text        NOT NULL,
    gauge_id    text,
    severity    text        NOT NULL,
    issued_at   timestamptz,
    valid_from  timestamptz,
    valid_until timestamptz,
    properties  jsonb       NOT NULL DEFAULT '{}',
    geom        geometry(MultiPolygon, 4326) NOT NULL,
    fetched_at  timestamptz NOT NULL
);
CREATE INDEX flood_areas_geom_idx ON flood_areas USING gist (geom);

-- +goose Down
DROP TABLE flood_areas;
DROP TABLE flood_gauges;
DROP TABLE wind_fields;
DROP TABLE weather_fields;
DROP TABLE weather_runs;
DROP TABLE weather_grid_nodes;
