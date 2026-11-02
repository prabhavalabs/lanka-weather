-- +goose Up
-- Official bulletins of the Disaster Management Centre, as listed by the
-- nuuuwan/lk_dmc indexes. Title and links are the source's own words; body is
-- only ever an excerpt copied unchanged from the PDF's extracted text, kept
-- when that text passes the cleanliness rules (ADR 0011).
CREATE TABLE official_bulletins (
    doc_id          text PRIMARY KEY,
    bulletin_type   text        NOT NULL CHECK (bulletin_type IN ('landslide', 'water_level', 'weather', 'situation')),
    title           text        NOT NULL,
    -- How clearly this is a warning, read from the title and the colour.
    severity        text        NOT NULL CHECK (severity IN ('warning', 'advisory', 'withdrawal', 'routine')),
    -- The colour the bulletin states for itself (Red, Amber, Yellow, Green).
    colour          text,
    message_id      text,
    issued_at       timestamptz NOT NULL,
    valid_from      timestamptz,
    valid_until     timestamptz,
    pdf_url         text        NOT NULL,
    source_page_url text        NOT NULL,
    lang            text        NOT NULL,
    body            text,
    text_status     text        NOT NULL CHECK (text_status IN ('pending', 'clean', 'unusable', 'unavailable')),
    -- Codes of districts the clean text names by exact word.
    districts       text[]      NOT NULL DEFAULT '{}',
    source          text        NOT NULL,
    -- When the upstream index was last rebuilt, as it reports.
    source_time     timestamptz NOT NULL,
    fetched_at      timestamptz NOT NULL,
    text_fetched_at timestamptz
);
CREATE INDEX official_bulletins_issued_idx ON official_bulletins (issued_at DESC);
CREATE INDEX official_bulletins_type_idx ON official_bulletins (bulletin_type, issued_at DESC);
CREATE INDEX official_bulletins_districts_idx ON official_bulletins USING gin (districts);

-- One NBRO landslide early-warning report per local date, and its level for
-- each divisional secretariat division. A division without a row has no
-- level in that report. ds_id is not a foreign key: reports are kept even
-- when a code from an older report no longer matches a region.
CREATE TABLE landslide_reports (
    report_date date PRIMARY KEY,
    source      text        NOT NULL,
    source_time timestamptz NOT NULL,
    fetched_at  timestamptz NOT NULL
);
CREATE TABLE landslide_levels (
    report_date date     NOT NULL REFERENCES landslide_reports (report_date) ON DELETE CASCADE,
    ds_id       text     NOT NULL,
    district_id text     NOT NULL,
    level       smallint NOT NULL CHECK (level BETWEEN 1 AND 3),
    PRIMARY KEY (report_date, ds_id)
);
CREATE INDEX landslide_levels_ds_idx ON landslide_levels (ds_id, report_date DESC);

-- USGS earthquake events, revised in place as USGS updates them.
CREATE TABLE earthquakes (
    id           text PRIMARY KEY,
    magnitude    double precision NOT NULL,
    mag_type     text             NOT NULL,
    place        text             NOT NULL,
    occurred_at  timestamptz      NOT NULL,
    lon          double precision NOT NULL,
    lat          double precision NOT NULL,
    depth_km     double precision NOT NULL,
    url          text             NOT NULL,
    alert        text,
    tsunami      boolean          NOT NULL,
    significance integer          NOT NULL,
    event_type   text             NOT NULL,
    status       text             NOT NULL,
    source       text             NOT NULL,
    source_time  timestamptz      NOT NULL,
    fetched_at   timestamptz      NOT NULL
);
CREATE INDEX earthquakes_occurred_idx ON earthquakes (occurred_at DESC);

-- GDACS tropical cyclone events in the North Indian Ocean. features is the
-- event's track, positions, wind buffers and uncertainty cone as a GeoJSON
-- feature list.
CREATE TABLE cyclones (
    event_id        bigint PRIMARY KEY,
    episode_id      integer          NOT NULL,
    name            text             NOT NULL,
    alert_level     text             NOT NULL,
    wind_kmh        double precision,
    severity_text   text             NOT NULL,
    lon             double precision NOT NULL,
    lat             double precision NOT NULL,
    country         text             NOT NULL,
    started_at      timestamptz      NOT NULL,
    latest_at       timestamptz      NOT NULL,
    advisory_source text             NOT NULL,
    report_url      text             NOT NULL,
    is_current      boolean          NOT NULL,
    features        jsonb            NOT NULL,
    source          text             NOT NULL,
    source_time     timestamptz      NOT NULL,
    fetched_at      timestamptz      NOT NULL
);
CREATE INDEX cyclones_current_idx ON cyclones (is_current, latest_at DESC);

-- +goose Down
DROP TABLE cyclones;
DROP TABLE earthquakes;
DROP TABLE landslide_levels;
DROP TABLE landslide_reports;
DROP TABLE official_bulletins;
