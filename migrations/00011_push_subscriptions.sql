-- +goose Up
-- Web Push subscriptions (ADR 0018). There are no accounts: the endpoint the
-- browser's push service issued is the identity, and its auth secret is the
-- proof of ownership for changes. Nothing else about the person is kept: no IP
-- address, no full user agent, and a "near me" point only as rounded to about
-- a kilometre by the API.
CREATE TABLE push_subscriptions (
    id                   bigserial PRIMARY KEY,
    endpoint             text        NOT NULL UNIQUE,
    -- The subscription's keys (RFC 8291), base64url as the browser gives them.
    p256dh               text        NOT NULL,
    auth                 text        NOT NULL,
    locale               text        NOT NULL CHECK (locale IN ('en', 'si', 'ta')),
    -- The browser family and platform only, such as "chrome-android".
    user_agent           text        NOT NULL,

    -- Preferences. Topics are the kinds of notice wanted; a notice is sent
    -- when its severity is at least min_severity and its area matches.
    topics               text[]      NOT NULL,
    min_severity         text        NOT NULL CHECK (min_severity IN ('info', 'watch', 'warning', 'danger')),
    whole_country        boolean     NOT NULL,
    -- District codes chosen from the regions table (LK-11).
    districts            text[]      NOT NULL DEFAULT '{}',
    -- "Near me": a point rounded to two decimals and a radius, and the
    -- districts within that radius, worked out when it is saved.
    near_lon             double precision,
    near_lat             double precision,
    near_radius_km       integer CHECK (near_radius_km BETWEEN 1 AND 100),
    near_districts       text[]      NOT NULL DEFAULT '{}',

    created_at           timestamptz NOT NULL,
    updated_at           timestamptz NOT NULL,
    -- Delivery health, for backoff and for dropping dead subscriptions.
    last_success_at      timestamptz,
    last_failure_at      timestamptz,
    success_count        integer     NOT NULL DEFAULT 0,
    failure_count        integer     NOT NULL DEFAULT 0,
    consecutive_failures integer     NOT NULL DEFAULT 0,
    -- No notice is attempted before this time (a 429, or backoff after errors).
    retry_after          timestamptz,
    last_test_at         timestamptz,
    CHECK ((near_lon IS NULL) = (near_lat IS NULL) AND (near_lon IS NULL) = (near_radius_km IS NULL))
);

-- Events the alert engine has raised: a warning newly in force, a gauge
-- rising across a level. The key is stable, so an event raised twice is
-- stored once. facts holds the values the message templates fill in.
CREATE TABLE push_events (
    dedupe_key  text PRIMARY KEY,
    kind        text        NOT NULL,
    -- The variant of its kind that picks its words: a level, a flood level.
    variant     text        NOT NULL,
    topic       text        NOT NULL,
    severity    text        NOT NULL CHECK (severity IN ('info', 'watch', 'warning', 'danger')),
    subject     text        NOT NULL,
    -- National events go to everyone with the topic, whatever their area.
    national    boolean     NOT NULL,
    districts   text[]      NOT NULL DEFAULT '{}',
    lon         double precision,
    lat         double precision,
    facts       jsonb       NOT NULL,
    url         text        NOT NULL,
    tag         text        NOT NULL,
    detected_at timestamptz NOT NULL,
    expires_at  timestamptz NOT NULL
);
CREATE INDEX push_events_detected_idx ON push_events (detected_at DESC);

-- What was sent to whom: one row per event per subscription, so nothing is
-- sent twice. Rows are claimed before a notice is sent and released if it
-- fails. Events batched into one notice share its notice_id.
CREATE SEQUENCE push_notice_ids;
CREATE TABLE push_deliveries (
    subscription_id bigint      NOT NULL REFERENCES push_subscriptions (id) ON DELETE CASCADE,
    event_key       text        NOT NULL REFERENCES push_events (dedupe_key) ON DELETE CASCADE,
    notice_id       bigint      NOT NULL,
    severity        text        NOT NULL,
    sent_at         timestamptz NOT NULL,
    PRIMARY KEY (subscription_id, event_key)
);
CREATE INDEX push_deliveries_sent_idx ON push_deliveries (sent_at DESC);
CREATE INDEX push_deliveries_notice_idx ON push_deliveries (notice_id);

-- The alert engine's memory of each subject (a gauge, a district's landslide
-- level): the level now, the level last alerted and since when the level has
-- been below it, for hysteresis.
CREATE TABLE push_alert_state (
    subject       text PRIMARY KEY,
    kind          text        NOT NULL,
    level         smallint    NOT NULL,
    alerted_level smallint    NOT NULL,
    alerted_at    timestamptz,
    below_since   timestamptz,
    updated_at    timestamptz NOT NULL
);

-- +goose Down
DROP TABLE push_alert_state;
DROP TABLE push_deliveries;
DROP SEQUENCE push_notice_ids;
DROP TABLE push_events;
DROP TABLE push_subscriptions;
