-- name: StartSourceRun :one
INSERT INTO source_runs (source, started_at, status)
VALUES ($1, $2, 'running')
RETURNING id;

-- name: FinishSourceRun :exec
UPDATE source_runs
SET finished_at = $2, status = $3, records = $4, error = $5
WHERE id = $1;

-- name: LastSuccessfulRun :one
SELECT id, source, started_at, finished_at, status, records, error
FROM source_runs
WHERE source = $1 AND status = 'ok'
ORDER BY started_at DESC
LIMIT 1;

-- name: LastRunStart :one
SELECT started_at
FROM source_runs
WHERE source = $1
ORDER BY started_at DESC
LIMIT 1;

-- name: AbandonRunningRuns :exec
-- Marks runs left 'running' by a stopped worker as failed.
UPDATE source_runs
SET status = 'error', finished_at = $2, error = 'abandoned: worker stopped during the run'
WHERE status = 'running' AND started_at < $1;

-- name: DeleteSourceRunsBefore :exec
DELETE FROM source_runs WHERE started_at < $1;
