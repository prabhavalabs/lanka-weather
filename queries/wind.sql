-- name: UpsertWindField :exec
INSERT INTO wind_fields (run_at, forecast_hour, valid_at, source, model, level, lon0, lat0, dlon, dlat, nx, ny,
                         scale, u, v, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (run_at, forecast_hour) DO UPDATE
SET valid_at = EXCLUDED.valid_at,
    source = EXCLUDED.source,
    model = EXCLUDED.model,
    level = EXCLUDED.level,
    lon0 = EXCLUDED.lon0,
    lat0 = EXCLUDED.lat0,
    dlon = EXCLUDED.dlon,
    dlat = EXCLUDED.dlat,
    nx = EXCLUDED.nx,
    ny = EXCLUDED.ny,
    scale = EXCLUDED.scale,
    u = EXCLUDED.u,
    v = EXCLUDED.v,
    fetched_at = EXCLUDED.fetched_at;

-- name: WindRunHours :many
SELECT forecast_hour FROM wind_fields WHERE run_at = $1 ORDER BY forecast_hour;

-- name: LatestWindRuns :many
-- Runs with the number of steps stored, newest first.
SELECT run_at, count(*)::integer AS steps, min(valid_at)::timestamptz AS first_valid_at,
       max(valid_at)::timestamptz AS last_valid_at, max(fetched_at)::timestamptz AS fetched_at
FROM wind_fields
GROUP BY run_at
ORDER BY run_at DESC
LIMIT $1;

-- name: WindField :one
SELECT run_at, forecast_hour, valid_at, source, model, level, lon0, lat0, dlon, dlat, nx, ny, scale, u, v, fetched_at
FROM wind_fields
WHERE run_at = $1 AND forecast_hour = $2;

-- name: DeleteWindRunsBefore :exec
DELETE FROM wind_fields WHERE run_at < $1;
