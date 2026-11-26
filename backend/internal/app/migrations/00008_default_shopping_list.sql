-- +goose Up
CREATE TABLE default_shopping_lists (
    user_id BLOB PRIMARY KEY NOT NULL,
    list_id TEXT NOT NULL,
    FOREIGN KEY (list_id, user_id) REFERENCES list_members(list_id, user_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE default_shopping_lists;
