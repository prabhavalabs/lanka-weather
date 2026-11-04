-- name: GetSourceVersion :one
SELECT source, version, source_time, fetched_at, records
FROM source_versions
WHERE source = $1;

-- name: PutSourceVersion :exec
INSERT INTO source_versions (source, version, source_time, fetched_at, records)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (source) DO UPDATE
SET version = EXCLUDED.version,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at,
    records = EXCLUDED.records;
