-- +goose Up
ALTER TABLE items ADD COLUMN new_unit TEXT CHECK (new_unit IN ('st', 'pkt', 'kg', 'g', 'l', 'dl', 'cl', 'ml', 'msk', 'tsk', 'krm'));
UPDATE items SET new_unit = CASE unit WHEN 'förp' THEN 'pkt' ELSE unit END WHERE unit IS NOT NULL;
ALTER TABLE items DROP COLUMN unit;
ALTER TABLE items RENAME COLUMN new_unit TO unit;

-- +goose Down
ALTER TABLE items ADD COLUMN old_unit TEXT CHECK (old_unit IN ('st', 'förp', 'kg', 'g', 'l', 'dl', 'cl', 'ml', 'msk', 'tsk', 'krm'));
UPDATE items SET old_unit = CASE unit WHEN 'pkt' THEN 'förp' ELSE unit END WHERE unit IS NOT NULL;
ALTER TABLE items DROP COLUMN unit;
ALTER TABLE items RENAME COLUMN old_unit TO unit;
