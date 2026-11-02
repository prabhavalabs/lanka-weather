-- name: DeleteCanalGauges :exec
DELETE FROM canal_gauges;

-- name: InsertCanalGauge :exec
INSERT INTO canal_gauges (id, name, canal_group, state, level_m, observed_at, change_3h_m, last_24h, source, source_time, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: ListCanalGauges :many
SELECT id, name, canal_group, state, level_m, observed_at, change_3h_m, last_24h, source, source_time, fetched_at
FROM canal_gauges
ORDER BY canal_group, name;

-- name: PutCanalContext :exec
INSERT INTO canal_context (id, credit, licence, upstream_url, sea_level_m, sea_trend, sea_observed_at, tide_note, tide, source, source_time, fetched_at)
VALUES (true, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE
SET credit = EXCLUDED.credit,
    licence = EXCLUDED.licence,
    upstream_url = EXCLUDED.upstream_url,
    sea_level_m = EXCLUDED.sea_level_m,
    sea_trend = EXCLUDED.sea_trend,
    sea_observed_at = EXCLUDED.sea_observed_at,
    tide_note = EXCLUDED.tide_note,
    tide = EXCLUDED.tide,
    source = EXCLUDED.source,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: GetCanalContext :one
SELECT credit, licence, upstream_url, sea_level_m, sea_trend, sea_observed_at, tide_note, tide, source, source_time, fetched_at
FROM canal_context
WHERE id;
