-- +goose Up
CREATE TABLE default_shopping_lists (
    user_id BLOB PRIMARY KEY NOT NULL,
    list_id TEXT NOT NULL,
    FOREIGN KEY (list_id, user_id) REFERENCES list_members(list_id, user_id) ON DELETE CASCADE
);

INSERT INTO default_shopping_lists (user_id, list_id)
SELECT membership.user_id, (
    SELECT lists.id FROM lists
    JOIN list_members ON list_members.list_id = lists.id
    WHERE list_members.user_id = membership.user_id
    ORDER BY lists.created_at, lists.id
    LIMIT 1
)
FROM (SELECT DISTINCT user_id FROM list_members) AS membership;

-- +goose Down
DROP TABLE default_shopping_lists;
