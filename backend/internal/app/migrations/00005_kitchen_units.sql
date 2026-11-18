-- +goose Up
ALTER TABLE items ADD COLUMN new_unit TEXT CHECK (new_unit IN ('st', 'förp', 'kg', 'g', 'l', 'dl', 'cl', 'ml', 'msk', 'tsk', 'krm'));
UPDATE items SET new_unit = unit WHERE unit IS NOT NULL;
ALTER TABLE items DROP COLUMN unit;
ALTER TABLE items RENAME COLUMN new_unit TO unit;

-- +goose Down
-- The old constraint rejects rollback while new units are in use, preserving their values.
ALTER TABLE items ADD COLUMN old_unit TEXT CHECK (old_unit IN ('st', 'förp', 'kg', 'g', 'l', 'dl', 'cl', 'ml'));
UPDATE items SET old_unit = unit WHERE unit IS NOT NULL;
ALTER TABLE items DROP COLUMN unit;
ALTER TABLE items RENAME COLUMN old_unit TO unit;
