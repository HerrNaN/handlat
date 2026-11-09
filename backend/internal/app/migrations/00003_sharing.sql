-- +goose Up
CREATE TABLE list_members (
    list_id TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    user_id BLOB NOT NULL REFERENCES users(id),
    PRIMARY KEY (list_id, user_id)
);
CREATE INDEX list_members_user ON list_members(user_id);
INSERT INTO list_members (list_id, user_id) SELECT id, owner_id FROM lists;
DROP INDEX lists_owner;
ALTER TABLE lists DROP COLUMN owner_id;

CREATE TABLE list_invitations (
    token_hash BLOB PRIMARY KEY NOT NULL,
    list_id TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL
);
CREATE INDEX list_invitations_expiry ON list_invitations(expires_at);
CREATE INDEX list_invitations_list ON list_invitations(list_id);

-- +goose Down
-- Returning to private lists assigns each list to one of its current members.
CREATE TABLE private_lists (
    id TEXT PRIMARY KEY NOT NULL,
    owner_id BLOB NOT NULL REFERENCES users(id),
    name TEXT NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
INSERT INTO private_lists
SELECT id, (SELECT user_id FROM list_members WHERE list_id = lists.id ORDER BY user_id LIMIT 1),
       name, created_at, updated_at FROM lists;
CREATE TABLE private_items (
    id TEXT NOT NULL,
    list_id TEXT NOT NULL REFERENCES private_lists(id) ON DELETE CASCADE,
    text TEXT NOT NULL CHECK (length(trim(text)) BETWEEN 1 AND 2000),
    department TEXT NOT NULL DEFAULT 'other',
    created_at INTEGER NOT NULL,
    checked_at INTEGER,
    deleted_at INTEGER,
    PRIMARY KEY (list_id, id)
);
INSERT INTO private_items SELECT * FROM items;
DROP TABLE list_invitations;
DROP TABLE list_members;
DROP TABLE items;
DROP TABLE lists;
ALTER TABLE private_lists RENAME TO lists;
ALTER TABLE private_items RENAME TO items;
CREATE INDEX lists_owner ON lists(owner_id);
CREATE INDEX items_list ON items(list_id);
