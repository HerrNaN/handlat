-- +goose Up
ALTER TABLE users ADD COLUMN recipes_enabled INTEGER NOT NULL DEFAULT 0 CHECK (recipes_enabled IN (0, 1));

CREATE TABLE recipe_collections (
    id INTEGER PRIMARY KEY,
    user_id BLOB NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE recipes (
    id TEXT PRIMARY KEY NOT NULL,
    collection_id INTEGER NOT NULL REFERENCES recipe_collections(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    portions INTEGER NOT NULL CHECK (portions BETWEEN 1 AND 9007199254740991),
    ingredients TEXT NOT NULL CHECK (json_valid(ingredients) AND json_type(ingredients) = 'array')
);

CREATE INDEX recipes_collection ON recipes(collection_id, name, id);

-- +goose Down
DROP TABLE recipes;
DROP TABLE recipe_collections;
ALTER TABLE users DROP COLUMN recipes_enabled;
