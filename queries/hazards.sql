-- name: UpsertEarthquake :exec
INSERT INTO earthquakes (id, magnitude, mag_type, place, occurred_at, lon, lat, depth_km, url, alert, tsunami,
                         significance, event_type, status, source, source_time, fetched_at)
VALUES (@id, @magnitude, @mag_type, @place, @occurred_at, @lon, @lat, @depth_km, @url, @alert, @tsunami,
        @significance, @event_type, @status, @source, @source_time, @fetched_at)
ON CONFLICT (id) DO UPDATE
SET magnitude = EXCLUDED.magnitude,
    mag_type = EXCLUDED.mag_type,
    place = EXCLUDED.place,
    occurred_at = EXCLUDED.occurred_at,
    lon = EXCLUDED.lon,
    lat = EXCLUDED.lat,
    depth_km = EXCLUDED.depth_km,
    url = EXCLUDED.url,
    alert = EXCLUDED.alert,
    tsunami = EXCLUDED.tsunami,
    significance = EXCLUDED.significance,
    event_type = EXCLUDED.event_type,
    status = EXCLUDED.status,
    source = EXCLUDED.source,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: DeleteEarthquake :exec
DELETE FROM earthquakes WHERE id = @id;

-- name: DeleteEarthquakesMissingSince :execrows
-- Events in the window USGS no longer returns: deleted, merged into another
-- event, or revised below the magnitude asked for.
DELETE FROM earthquakes
WHERE occurred_at >= @since
  AND NOT (id = ANY (@keep::text[]));

-- name: DeleteEarthquakesBefore :exec
DELETE FROM earthquakes WHERE occurred_at < @before;

-- name: ListEarthquakes :many
SELECT id, magnitude, mag_type, place, occurred_at, lon, lat, depth_km, url, alert, tsunami, significance, event_type,
       status, source, source_time, fetched_at
FROM earthquakes
WHERE occurred_at >= @since
ORDER BY occurred_at DESC, id;

-- name: UpsertCyclone :exec
INSERT INTO cyclones (event_id, episode_id, name, alert_level, wind_kmh, severity_text, lon, lat, country, started_at,
                      latest_at, advisory_source, report_url, is_current, features, source, source_time, fetched_at)
VALUES (@event_id, @episode_id, @name, @alert_level, @wind_kmh, @severity_text, @lon, @lat, @country, @started_at,
        @latest_at, @advisory_source, @report_url, @is_current, @features, @source, @source_time, @fetched_at)
ON CONFLICT (event_id) DO UPDATE
SET episode_id = EXCLUDED.episode_id,
    name = EXCLUDED.name,
    alert_level = EXCLUDED.alert_level,
    wind_kmh = EXCLUDED.wind_kmh,
    severity_text = EXCLUDED.severity_text,
    lon = EXCLUDED.lon,
    lat = EXCLUDED.lat,
    country = EXCLUDED.country,
    started_at = EXCLUDED.started_at,
    latest_at = EXCLUDED.latest_at,
    advisory_source = EXCLUDED.advisory_source,
    report_url = EXCLUDED.report_url,
    is_current = EXCLUDED.is_current,
    features = CASE WHEN EXCLUDED.features = '[]'::jsonb THEN cyclones.features ELSE EXCLUDED.features END,
    source = EXCLUDED.source,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: ClearCurrentCyclones :exec
-- Events the list no longer calls current in the basin.
UPDATE cyclones SET is_current = false WHERE is_current AND NOT (event_id = ANY (@keep::bigint[]));

-- name: ListCurrentCyclones :many
SELECT event_id, episode_id, name, alert_level, wind_kmh, severity_text, lon, lat, country, started_at, latest_at,
       advisory_source, report_url, is_current, features, source, source_time, fetched_at
FROM cyclones
WHERE is_current
ORDER BY latest_at DESC, event_id;
