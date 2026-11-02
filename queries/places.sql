-- name: ClearPlaceStaging :exec
TRUNCATE place_staging;

-- name: StagePlaces :copyfrom
INSERT INTO place_staging (id, name, name_si, name_ta, alt_names, kind, population, lon, lat)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: MergeStagedPlaces :execrows
INSERT INTO places (id, name, name_si, name_ta, alt_names, kind, population, lon, lat, source, source_time, fetched_at)
SELECT id, name, name_si, name_ta, alt_names, kind, population, lon, lat,
       @source::text, @source_time::timestamptz, @fetched_at::timestamptz
FROM place_staging
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    name_si = EXCLUDED.name_si,
    name_ta = EXCLUDED.name_ta,
    alt_names = EXCLUDED.alt_names,
    kind = EXCLUDED.kind,
    population = EXCLUDED.population,
    lon = EXCLUDED.lon,
    lat = EXCLUDED.lat,
    source = EXCLUDED.source,
    source_time = EXCLUDED.source_time,
    fetched_at = EXCLUDED.fetched_at;

-- name: DeletePlacesNotStaged :execrows
DELETE FROM places p
WHERE NOT EXISTS (SELECT 1 FROM place_staging s WHERE s.id = p.id);

-- name: AssignPlaceRegions :execrows
-- Sets each place's GN division: the one containing it or, for a place just
-- off the simplified coastline, the nearest within about 3 km.
UPDATE places p
SET gn_id = found.gn_id
FROM (
    SELECT p2.id,
           coalesce(
               (SELECT r.id FROM regions r
                WHERE r.level = 'gn' AND ST_Intersects(r.geom, p2.geom)
                ORDER BY r.id LIMIT 1),
               (SELECT r.id FROM regions r
                WHERE r.level = 'gn' AND ST_DWithin(r.geom, p2.geom, 0.03)
                ORDER BY ST_Distance(r.geom, p2.geom), r.id LIMIT 1)
           ) AS gn_id
    FROM places p2
) found
WHERE found.id = p.id AND p.gn_id IS DISTINCT FROM found.gn_id;

-- name: InsertPlaceSearchNames :execrows
-- Indexes each place's names. Importance follows OpenStreetMap's place
-- hierarchy (city > town > suburb > village > hamlet), then population.
INSERT INTO search_names (target_kind, target_id, name, lang, folded, importance)
SELECT 'place', p.id, n.name, n.lang, search_fold(n.name),
       (CASE p.kind
            WHEN 'city' THEN 1.0
            WHEN 'town' THEN 0.75
            WHEN 'suburb' THEN 0.55
            WHEN 'village' THEN 0.45
            WHEN 'quarter' THEN 0.4
            WHEN 'neighbourhood' THEN 0.35
            WHEN 'hamlet' THEN 0.25
            WHEN 'locality' THEN 0.2
            ELSE 0.1
        END + 0.1 * least(1.0, ln(greatest(coalesce(p.population, 1), 1)) / ln(10000000)))::real
FROM places p
CROSS JOIN LATERAL (
    SELECT p.name, 'en'
    UNION ALL SELECT p.name_si, 'si'
    UNION ALL SELECT p.name_ta, 'ta'
    UNION ALL SELECT unnest(p.alt_names), 'alt'
) AS n(name, lang)
WHERE n.name IS NOT NULL AND search_fold(n.name) <> ''
ON CONFLICT DO NOTHING;

-- name: CountPlaces :one
SELECT count(*)::int FROM places;

-- name: SetWordSimilarityThreshold :exec
-- Lowers the trigram threshold of the <% operator for the transaction, so
-- one or two typos in a short word still match ("galel" for "galle").
SELECT set_config('pg_trgm.word_similarity_threshold', @threshold::text, true);

-- name: SearchNames :many
-- Scores each name's match: 1 when equal; 0.8 to 0.9 when the query starts
-- the name, and 0.7 to 0.8 when it starts a later word, more the more of the
-- name it covers; otherwise 0.8 times the trigram word similarity (typos).
-- Keeps each target's best name and ranks by 60% match and 40% importance,
-- so a slightly misspelt city still comes before a village whose name the
-- query happens to start. fuzzy enables the trigram match; it is off for
-- queries too short to have trigrams.
WITH input AS (SELECT search_fold(@query::text) AS q),
matches AS (
    SELECT DISTINCT ON (n.target_kind, n.target_id)
           n.target_kind, n.target_id, n.name AS matched_name, n.importance,
           (CASE
                WHEN n.folded = input.q THEN 1.0
                WHEN n.folded LIKE input.q || '%'
                    THEN 0.8 + 0.1 * length(input.q) / greatest(length(n.folded), 1)
                WHEN n.folded LIKE '% ' || input.q || '%'
                    THEN 0.7 + 0.1 * length(input.q) / greatest(length(n.folded), 1)
                ELSE 0.8 * word_similarity(input.q, n.folded)
            END)::float8 AS match
    FROM search_names n, input
    WHERE input.q <> ''
      AND (n.folded LIKE input.q || '%' OR (@fuzzy::bool AND input.q <% n.folded))
    ORDER BY n.target_kind, n.target_id, match DESC, length(n.name), n.name
),
ranked AS (
    SELECT m.target_kind, m.target_id, m.matched_name,
           (m.match * 0.6 + m.importance * 0.4)::float8 AS score
    FROM matches m
    ORDER BY score DESC, length(m.matched_name), m.matched_name, m.target_id
    LIMIT @max_results::int
)
SELECT ranked.target_kind, ranked.target_id, ranked.matched_name, ranked.score,
       coalesce(r.name, p.name)::text AS name,
       coalesce(r.name_si, p.name_si) AS name_si,
       coalesce(r.name_ta, p.name_ta) AS name_ta,
       coalesce(r.level, p.kind)::text AS kind,
       ST_X(coalesce(r.label_point, p.geom))::float8 AS lon,
       ST_Y(coalesce(r.label_point, p.geom))::float8 AS lat,
       r.west, r.south, r.east, r.north,
       coalesce(r.population, p.population) AS population,
       prov.id AS province_id, prov.name AS province_name,
       dist.id AS district_id, dist.name AS district_name,
       dsr.id AS ds_id, dsr.name AS ds_name,
       pg.id AS gn_id, pg.name AS gn_name
FROM ranked
LEFT JOIN regions r ON ranked.target_kind = 'region' AND r.id = ranked.target_id
LEFT JOIN places p ON ranked.target_kind = 'place' AND p.id = ranked.target_id
LEFT JOIN regions pg ON pg.id = p.gn_id
LEFT JOIN regions prov ON prov.id = coalesce(r.province_id, pg.province_id)
LEFT JOIN regions dist ON dist.id = coalesce(r.district_id, pg.district_id)
LEFT JOIN regions dsr ON dsr.id = coalesce(r.ds_id, pg.ds_id)
ORDER BY ranked.score DESC, length(ranked.matched_name), ranked.matched_name, ranked.target_id;
