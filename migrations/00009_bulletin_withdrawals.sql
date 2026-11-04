-- +goose Up
-- The basins a flood warning withdrawal notice lists, as the issuer spells them.
-- Null means the notice's text has not been read for them, empty that it has and
-- no list could be followed, so its bulletin is shown beside earlier warnings
-- without ending any.
ALTER TABLE official_bulletins ADD COLUMN withdrawn_basins text[];

-- +goose Down
ALTER TABLE official_bulletins DROP COLUMN withdrawn_basins;
