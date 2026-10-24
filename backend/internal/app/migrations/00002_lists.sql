-- +goose Up
CREATE TABLE lists (
    id TEXT PRIMARY KEY NOT NULL,
    owner_id BLOB NOT NULL REFERENCES users(id),
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX lists_owner ON lists(owner_id);

CREATE TABLE items (
    id TEXT NOT NULL,
    list_id TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    text TEXT NOT NULL CHECK (length(trim(text)) BETWEEN 1 AND 2000),
    department TEXT NOT NULL DEFAULT 'other',
    created_at TEXT NOT NULL,
    checked_at TEXT,
    deleted_at TEXT,
    PRIMARY KEY (list_id, id)
);
CREATE INDEX items_list ON items(list_id);

-- +goose Down
DROP TABLE items;
DROP TABLE lists;
