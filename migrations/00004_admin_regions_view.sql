-- +goose Up
-- The administrative regions as the weather read side reads them
-- (internal/weather.PostGISRegions, docs/api/weather-and-floods.md): one row
-- per region with its level, name, ancestors, area, a point inside it and
-- its boundary. The view decouples the weather aggregation from the columns
-- of the regions table (migration 00002), which serves boundaries and
-- search as well.
CREATE VIEW admin_regions AS
SELECT id,
       level,
       name,
       parent_id,
       province_id,
       district_id,
       ds_id,
       area_km2,
       label_point,
       geom
FROM regions;

-- +goose Down
DROP VIEW admin_regions;
