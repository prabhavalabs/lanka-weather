-- +goose Up
-- Live rain gauges from HMIS, through the copy Sri Lanka Water Levels publishes
-- (ADR 0015). A reading is a rolling 24-hour sum, so nothing shorter can be
-- derived from it: it is stored as published.
CREATE TABLE rain_gauges (
    id                 text PRIMARY KEY,
    -- The device's name in the feed.
    device             text             NOT NULL,
    name               text             NOT NULL,
    institution        text             NOT NULL,
    lon                double precision NOT NULL,
    lat                double precision NOT NULL,
    latest_observed_at timestamptz,
    latest_rain_mm     double precision,
    source             text             NOT NULL,
    source_time        timestamptz      NOT NULL,
    fetched_at         timestamptz      NOT NULL
);

CREATE TABLE rain_gauge_readings (
    gauge_id    text             NOT NULL REFERENCES rain_gauges (id) ON DELETE CASCADE,
    observed_at timestamptz      NOT NULL,
    rain_mm     double precision NOT NULL,
    source      text             NOT NULL,
    source_time timestamptz      NOT NULL,
    fetched_at  timestamptz      NOT NULL,
    PRIMARY KEY (gauge_id, observed_at)
);
CREATE INDEX rain_gauge_readings_observed_idx ON rain_gauge_readings (observed_at);

-- Colombo's canal gauges, replaced whole on each fetch. The feed gives no
-- coordinates and none are guessed, so they have no position here. A gauge's
-- state stays in the source's own words.
CREATE TABLE canal_gauges (
    id          text PRIMARY KEY,
    name        text        NOT NULL,
    canal_group text        NOT NULL,
    state       text        NOT NULL,
    level_m     double precision,
    observed_at timestamptz,
    change_3h_m double precision,
    last_24h    jsonb       NOT NULL,
    source      text        NOT NULL,
    source_time timestamptz NOT NULL,
    fetched_at  timestamptz NOT NULL
);

-- What came with the gauges: the credit and licence the source asks for, the sea
-- level at the Kelani mouth and the predicted tide. One row.
CREATE TABLE canal_context (
    id              boolean PRIMARY KEY DEFAULT true CHECK (id),
    credit          text        NOT NULL,
    licence         text        NOT NULL,
    upstream_url    text        NOT NULL,
    sea_level_m     double precision,
    sea_trend       text,
    sea_observed_at timestamptz,
    tide_note       text        NOT NULL,
    tide            jsonb       NOT NULL,
    source          text        NOT NULL,
    source_time     timestamptz NOT NULL,
    fetched_at      timestamptz NOT NULL
);

-- +goose Down
DROP TABLE canal_context;
DROP TABLE canal_gauges;
DROP TABLE rain_gauge_readings;
DROP TABLE rain_gauges;
