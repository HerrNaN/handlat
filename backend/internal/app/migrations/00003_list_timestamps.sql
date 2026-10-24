-- +goose Up
CREATE TABLE lists_new (
    id TEXT PRIMARY KEY NOT NULL,
    owner_id BLOB NOT NULL REFERENCES users(id),
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE items_new (
    id TEXT NOT NULL,
    list_id TEXT NOT NULL REFERENCES lists_new(id) ON DELETE CASCADE,
    text TEXT NOT NULL CHECK (length(trim(text)) BETWEEN 1 AND 2000),
    department TEXT NOT NULL DEFAULT 'other',
    created_at INTEGER NOT NULL,
    checked_at INTEGER,
    deleted_at INTEGER,
    PRIMARY KEY (list_id, id)
);

INSERT INTO lists_new
SELECT id, owner_id, name, unixepoch(created_at), unixepoch(updated_at) FROM lists;
INSERT INTO items_new
SELECT id, list_id, text, department, unixepoch(created_at), unixepoch(checked_at), unixepoch(deleted_at) FROM items;

DROP TABLE items;
DROP TABLE lists;
ALTER TABLE lists_new RENAME TO lists;
ALTER TABLE items_new RENAME TO items;
CREATE INDEX lists_owner ON lists(owner_id);
CREATE INDEX items_list ON items(list_id);

-- +goose Down
CREATE TABLE lists_old (
    id TEXT PRIMARY KEY NOT NULL,
    owner_id BLOB NOT NULL REFERENCES users(id),
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE items_old (
    id TEXT NOT NULL,
    list_id TEXT NOT NULL REFERENCES lists_old(id) ON DELETE CASCADE,
    text TEXT NOT NULL CHECK (length(trim(text)) BETWEEN 1 AND 2000),
    department TEXT NOT NULL DEFAULT 'other',
    created_at TEXT NOT NULL,
    checked_at TEXT,
    deleted_at TEXT,
    PRIMARY KEY (list_id, id)
);

INSERT INTO lists_old (id, owner_id, name, created_at, updated_at)
SELECT id, owner_id, name,
       strftime('%Y-%m-%dT%H:%M:%SZ', created_at, 'unixepoch'),
       strftime('%Y-%m-%dT%H:%M:%SZ', updated_at, 'unixepoch') FROM lists;
INSERT INTO items_old
SELECT id, list_id, text, department,
       strftime('%Y-%m-%dT%H:%M:%SZ', created_at, 'unixepoch'),
       strftime('%Y-%m-%dT%H:%M:%SZ', checked_at, 'unixepoch'),
       strftime('%Y-%m-%dT%H:%M:%SZ', deleted_at, 'unixepoch') FROM items;

DROP TABLE items;
DROP TABLE lists;
ALTER TABLE lists_old RENAME TO lists;
ALTER TABLE items_old RENAME TO items;
CREATE INDEX lists_owner ON lists(owner_id);
CREATE INDEX items_list ON items(list_id);
