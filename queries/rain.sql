-- name: UpsertRainTile :batchexec
INSERT INTO rain_tiles (valid_at, z, x, y, png, source, fetched_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (valid_at, z, x, y) DO UPDATE
SET png = EXCLUDED.png,
    source = EXCLUDED.source,
    fetched_at = EXCLUDED.fetched_at;

-- name: RainSteps :many
-- The stored time steps, newest first, with how many tiles each holds and how
-- many of those are empty.
SELECT valid_at,
       count(*)::integer                       AS tiles,
       (count(*) - count(png))::integer        AS empty_tiles,
       max(fetched_at)::timestamptz            AS fetched_at
FROM rain_tiles
GROUP BY valid_at
ORDER BY valid_at DESC;

-- name: RainTile :one
SELECT png, source, fetched_at
FROM rain_tiles
WHERE valid_at = $1 AND z = $2 AND x = $3 AND y = $4;

-- name: NewestRainStep :one
SELECT valid_at FROM rain_tiles ORDER BY valid_at DESC LIMIT 1;

-- name: DeleteRainTilesBefore :exec
DELETE FROM rain_tiles WHERE valid_at < $1;
