-- +goose Up
-- When an area was first stored, kept across the collectors' replacements, so
-- the delay between Google's issue time and our first sight of an area can be
-- shown and measured. Areas stored before this keep their last fetch time.
ALTER TABLE flood_areas ADD COLUMN first_seen_at timestamptz;
UPDATE flood_areas SET first_seen_at = fetched_at;
ALTER TABLE flood_areas ALTER COLUMN first_seen_at SET NOT NULL;
CREATE INDEX flood_areas_kind_idx ON flood_areas (kind);

-- +goose Down
DROP INDEX flood_areas_kind_idx;
ALTER TABLE flood_areas DROP COLUMN first_seen_at;
