-- name: UpsertWeatherGridNode :batchexec
-- The cell is the square of one grid step centred on the node.
INSERT INTO weather_grid_nodes (grid, node, lon, lat, land, elevation_m, cell, source, updated_at)
VALUES (sqlc.arg(grid), sqlc.arg(node), sqlc.arg(lon), sqlc.arg(lat), sqlc.arg(land), sqlc.narg(elevation_m),
        ST_MakeEnvelope(sqlc.arg(west)::float8, sqlc.arg(south)::float8, sqlc.arg(east)::float8,
                        sqlc.arg(north)::float8, 4326),
        sqlc.arg(source), sqlc.arg(updated_at))
ON CONFLICT (grid, node) DO UPDATE
SET lon = EXCLUDED.lon,
    lat = EXCLUDED.lat,
    land = EXCLUDED.land,
    elevation_m = COALESCE(EXCLUDED.elevation_m, weather_grid_nodes.elevation_m),
    cell = EXCLUDED.cell,
    source = EXCLUDED.source,
    updated_at = EXCLUDED.updated_at;

-- name: InsertWeatherRun :one
INSERT INTO weather_runs (kind, source, model, grid, nodes, source_time, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: InsertWeatherField :batchexec
INSERT INTO weather_fields (run_id, variable, source, valid_from, step_seconds, steps, node_values, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: LatestWeatherRun :one
SELECT id, kind, source, model, grid, nodes, source_time, fetched_at
FROM weather_runs
WHERE kind = $1
ORDER BY id DESC
LIMIT 1;

-- name: LatestWeatherRunIDs :one
-- The newest run of each kind, 0 when there is none; checked before
-- re-reading a run.
SELECT COALESCE((SELECT max(id) FROM weather_runs WHERE kind = 'current'), 0)::bigint AS current_id,
       COALESCE((SELECT max(id) FROM weather_runs WHERE kind = 'forecast'), 0)::bigint AS forecast_id;

-- name: WeatherRunFields :many
SELECT variable, source, valid_from, step_seconds, steps, node_values, fetched_at
FROM weather_fields
WHERE run_id = $1
ORDER BY variable;

-- name: DeleteWeatherRunsBefore :exec
-- Old runs go, but the newest of each kind always stays, so a source outage
-- shows old data marked stale rather than none.
DELETE FROM weather_runs AS old
WHERE old.fetched_at < @before
  AND old.id NOT IN (SELECT max(newest.id) FROM weather_runs AS newest GROUP BY newest.kind);
