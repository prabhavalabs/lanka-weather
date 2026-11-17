-- +goose Up
-- The app sent its stored list of topics on every start, so a subscription
-- given heavy_rain (00014) or flood_risk (00015) on the server lost them the
-- next time the app opened (November 2026). The app now adds topics it has
-- not shown the person before; this gives them back to subscriptions that
-- lost them, on the same rule as those migrations.
UPDATE push_subscriptions
SET topics = array_append(topics, 'heavy_rain')
WHERE 'flood_forecast' = ANY(topics) AND NOT 'heavy_rain' = ANY(topics);

UPDATE push_subscriptions
SET topics = array_append(topics, 'flood_risk')
WHERE 'flood_forecast' = ANY(topics) AND NOT 'flood_risk' = ANY(topics);

-- +goose Down
-- Nothing to undo: the topics are wanted, and the settings can turn them off.
SELECT 1;
