-- +goose Up
-- The combined flood-risk estimate is a new topic, on by default. As with
-- heavy rain (00014), a subscription that chose flood forecasts gets it; the
-- settings can turn it off.
UPDATE push_subscriptions
SET topics = array_append(topics, 'flood_risk')
WHERE 'flood_forecast' = ANY(topics) AND NOT 'flood_risk' = ANY(topics);

-- +goose Down
UPDATE push_subscriptions SET topics = array_remove(topics, 'flood_risk');
