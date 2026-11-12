-- +goose Up
ALTER TABLE items ADD COLUMN amount TEXT;
ALTER TABLE items ADD COLUMN unit TEXT CHECK (unit IN ('st', 'förp', 'kg', 'g', 'l', 'dl', 'cl', 'ml'));

-- +goose Down
ALTER TABLE items DROP COLUMN unit;
ALTER TABLE items DROP COLUMN amount;
