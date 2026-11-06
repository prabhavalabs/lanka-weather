-- +goose Up
-- Heavy rain measured at the live gauges is a new topic, on by default. A
-- subscription that wanted flood forecasts could not have chosen it, and
-- wants it for the same reason, so it is added there; the settings can turn
-- it off.
UPDATE push_subscriptions
SET topics = array_append(topics, 'heavy_rain')
WHERE 'flood_forecast' = ANY(topics) AND NOT 'heavy_rain' = ANY(topics);

-- +goose Down
UPDATE push_subscriptions SET topics = array_remove(topics, 'heavy_rain');
