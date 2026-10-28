-- +goose Up
CREATE TABLE lists (
    id TEXT PRIMARY KEY NOT NULL,
    owner_id BLOB NOT NULL REFERENCES users(id),
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX lists_owner ON lists(owner_id);

CREATE TABLE items (
    id TEXT NOT NULL,
    list_id TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    text TEXT NOT NULL CHECK (length(trim(text)) BETWEEN 1 AND 2000),
    department TEXT NOT NULL DEFAULT 'other',
    created_at INTEGER NOT NULL,
    checked_at INTEGER,
    deleted_at INTEGER,
    PRIMARY KEY (list_id, id)
);
CREATE INDEX items_list ON items(list_id);

-- +goose Down
DROP TABLE items;
DROP TABLE lists;
