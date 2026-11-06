-- +goose Up
-- Flash-flood alerts follow a district rather than one of Google's areas,
-- which Google reissues under a new ID every few hours. Each district takes
-- the state of the areas it was alerted for, the highest level and the latest
-- first, so a forecast already alerted is not sent again on deploy. The old
-- per-area states remain and fall away as the engine no longer sees them.
INSERT INTO push_alert_state (subject, kind, level, alerted_level, alerted_at, below_since, updated_at)
SELECT DISTINCT ON (district)
       'flash:district:' || district, 'flash_flood', s.level, s.alerted_level, s.alerted_at, s.below_since, s.updated_at
FROM push_alert_state s
JOIN push_events e ON e.subject = s.subject
CROSS JOIN LATERAL unnest(e.districts) AS district
WHERE s.kind = 'flash_flood'
  AND s.subject NOT LIKE 'flash:district:%'
ORDER BY district, s.alerted_level DESC, s.level DESC, s.updated_at DESC
ON CONFLICT (subject) DO NOTHING;

-- +goose Down
DELETE FROM push_alert_state WHERE subject LIKE 'flash:district:%';
