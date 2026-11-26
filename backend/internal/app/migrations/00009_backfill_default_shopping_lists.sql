-- +goose Up
INSERT INTO default_shopping_lists (user_id, list_id)
SELECT membership.user_id, (
    SELECT lists.id FROM lists
    JOIN list_members ON list_members.list_id = lists.id
    WHERE list_members.user_id = membership.user_id
    ORDER BY lists.created_at, lists.id
    LIMIT 1
)
FROM (SELECT DISTINCT user_id FROM list_members) AS membership
WHERE NOT EXISTS (
    SELECT 1 FROM default_shopping_lists WHERE user_id = membership.user_id
);

-- +goose Down
SELECT 1;
