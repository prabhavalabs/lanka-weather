-- name: GetPushSubscriptionByEndpoint :one
SELECT * FROM push_subscriptions WHERE endpoint = @endpoint;

-- name: InsertPushSubscription :one
INSERT INTO push_subscriptions (endpoint, p256dh, auth, locale, user_agent, topics, min_severity, whole_country,
                                districts, near_lon, near_lat, near_radius_km, near_districts, created_at, updated_at)
VALUES (@endpoint, @p256dh, @auth, @locale, @user_agent, @topics, @min_severity, @whole_country,
        @districts, @near_lon, @near_lat, @near_radius_km, @near_districts, @now, @now)
RETURNING *;

-- name: UpdatePushPreferences :one
UPDATE push_subscriptions
SET locale = @locale,
    user_agent = @user_agent,
    topics = @topics,
    min_severity = @min_severity,
    whole_country = @whole_country,
    districts = @districts,
    near_lon = @near_lon,
    near_lat = @near_lat,
    near_radius_km = @near_radius_km,
    near_districts = @near_districts,
    updated_at = @now
WHERE id = @id
RETURNING *;

-- name: RenewPushSubscription :one
-- Moves a subscription to the endpoint and keys its push service issued in
-- place of the old ones, keeping its preferences.
UPDATE push_subscriptions
SET endpoint = @endpoint,
    p256dh = @p256dh,
    auth = @auth,
    updated_at = @now,
    consecutive_failures = 0,
    retry_after = NULL
WHERE id = @id
RETURNING *;

-- name: DeletePushSubscription :execrows
DELETE FROM push_subscriptions WHERE id = @id;

-- name: ClaimPushTest :execrows
-- Records a test notice unless one was sent too recently.
UPDATE push_subscriptions
SET last_test_at = @now::timestamptz
WHERE id = @id
  AND (last_test_at IS NULL OR last_test_at < @not_before::timestamptz);

-- name: ListPushSubscriptions :many
SELECT * FROM push_subscriptions ORDER BY id;

-- name: CountPushSubscriptions :one
SELECT count(*)::int FROM push_subscriptions;

-- name: RecordPushSuccess :exec
UPDATE push_subscriptions
SET last_success_at = @at::timestamptz,
    success_count = success_count + 1,
    consecutive_failures = 0,
    retry_after = NULL
WHERE id = @id;

-- name: RecordPushFailure :exec
UPDATE push_subscriptions
SET last_failure_at = @at::timestamptz,
    failure_count = failure_count + 1,
    consecutive_failures = consecutive_failures + 1,
    retry_after = @retry_after
WHERE id = @id;

-- name: DeleteFailingPushSubscriptions :execrows
-- Drops subscriptions that have failed many times in a row and not succeeded
-- for a long time: their push service no longer accepts them.
DELETE FROM push_subscriptions
WHERE consecutive_failures >= @max_failures
  AND coalesce(last_success_at, created_at) < @before::timestamptz;

-- name: InsertPushEvent :execrows
INSERT INTO push_events (dedupe_key, kind, variant, topic, severity, subject, national, districts, lon, lat, facts, url,
                         tag, detected_at, expires_at)
VALUES (@dedupe_key, @kind, @variant, @topic, @severity, @subject, @national, @districts, @lon, @lat, @facts, @url,
        @tag, @detected_at, @expires_at)
ON CONFLICT (dedupe_key) DO NOTHING;

-- name: ListActivePushEvents :many
SELECT * FROM push_events
WHERE expires_at > @now
ORDER BY detected_at, dedupe_key;

-- name: ListPushDeliveriesFor :many
SELECT subscription_id, event_key FROM push_deliveries WHERE event_key = ANY(@keys::text[]);

-- name: RecentPushNoticeCounts :many
-- Notices other than danger sent to each subscription since a time, for the
-- hourly cap.
SELECT subscription_id, count(DISTINCT notice_id)::int AS notices
FROM push_deliveries
WHERE sent_at > @since
  AND severity <> 'danger'
GROUP BY subscription_id;

-- name: NextPushNoticeID :one
SELECT nextval('push_notice_ids')::bigint;

-- name: ClaimPushDelivery :execrows
INSERT INTO push_deliveries (subscription_id, event_key, notice_id, severity, sent_at)
VALUES (@subscription_id, @event_key, @notice_id, @severity, @sent_at)
ON CONFLICT (subscription_id, event_key) DO NOTHING;

-- name: ReleasePushNotice :exec
DELETE FROM push_deliveries WHERE notice_id = @notice_id;

-- name: ListPushAlertState :many
SELECT * FROM push_alert_state ORDER BY subject;

-- name: UpsertPushAlertState :exec
INSERT INTO push_alert_state (subject, kind, level, alerted_level, alerted_at, below_since, updated_at)
VALUES (@subject, @kind, @level, @alerted_level, @alerted_at, @below_since, @updated_at)
ON CONFLICT (subject) DO UPDATE
SET kind = EXCLUDED.kind,
    level = EXCLUDED.level,
    alerted_level = EXCLUDED.alerted_level,
    alerted_at = EXCLUDED.alerted_at,
    below_since = EXCLUDED.below_since,
    updated_at = EXCLUDED.updated_at;

-- name: DeletePushAlertState :exec
DELETE FROM push_alert_state WHERE subject = @subject;

-- name: DeletePushEventsBefore :execrows
-- Retention: events, and with them their deliveries, are kept 30 days.
DELETE FROM push_events WHERE detected_at < @before;

-- name: ListDistrictNames :many
SELECT id, name, name_si, name_ta FROM regions WHERE level = 'district' ORDER BY id;

-- name: DistrictsWithin :many
-- Districts within a distance of a point, for "near me".
SELECT id FROM regions
WHERE level = 'district'
  AND ST_DWithin(geom::geography, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326)::geography, @meters::float8)
ORDER BY id;

-- name: DistrictAt :many
-- The district containing a point, or the nearest within about 5 km, for a
-- gauge on a river that forms a boundary or a point just off the coast.
SELECT id FROM regions
WHERE level = 'district'
  AND ST_DWithin(geom, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326), 0.05)
ORDER BY ST_Distance(geom, ST_SetSRID(ST_MakePoint(@lon::float8, @lat::float8), 4326)), id
LIMIT 1;

-- name: FloodAreaDistricts :many
SELECT r.id FROM regions r
JOIN flood_areas a ON ST_Intersects(r.geom, a.geom)
WHERE a.id = @area_id AND r.level = 'district'
ORDER BY r.id;

-- name: BasinDistrictShares :many
-- The share of each named river basin in each district it crosses, so a flood
-- warning that names a basin can name the districts it covers.
SELECT b.name::text AS basin, r.id::text AS district,
       (ST_Area(ST_Intersection(b.geom, r.geom_simple)::geography)
        / NULLIF(ST_Area(b.geom::geography), 0))::float8 AS share
FROM official_basins b
JOIN regions r ON r.level = 'district' AND ST_Intersects(b.geom, r.geom_simple)
WHERE b.named
ORDER BY b.name, share DESC;

-- name: PushInbox :many
-- What was sent to one subscription since a time, newest notice first: every
-- event of each notice, for the in-app list of notices.
SELECT d.notice_id, d.sent_at, d.severity AS notice_severity, e.*
FROM push_deliveries d
JOIN push_events e ON e.dedupe_key = d.event_key
WHERE d.subscription_id = @subscription_id AND d.sent_at >= @since
ORDER BY d.sent_at DESC, d.notice_id DESC, e.detected_at DESC
LIMIT @max_rows;
