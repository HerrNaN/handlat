-- name: CreateList :exec
INSERT INTO lists (id, owner_id, name, created_at, updated_at)
VALUES (@id, @owner_id, @name, @now, @now);

-- name: GetList :one
SELECT * FROM lists WHERE id = @id;

-- name: ListLists :many
SELECT lists.id, lists.name, count(items.id) AS item_count
FROM lists LEFT JOIN items ON items.list_id = lists.id AND items.deleted_at IS NULL
WHERE lists.owner_id = @owner_id
GROUP BY lists.id ORDER BY lists.created_at, lists.id;

-- name: TouchList :exec
UPDATE lists SET updated_at = @now
WHERE id = @id;

-- name: RenameList :exec
UPDATE lists SET name = @name WHERE id = @id;

-- name: ListItems :many
SELECT * FROM items WHERE list_id = @list_id AND deleted_at IS NULL ORDER BY created_at, id;

-- name: AddItem :exec
INSERT INTO items (id, list_id, text, department, created_at)
VALUES (@id, @list_id, @text, @department, @created_at)
ON CONFLICT (list_id, id) DO NOTHING;

-- name: UpdateItem :exec
-- narg makes omitted PATCH fields nullable parameters despite NOT NULL columns.
-- COALESCE preserves the current value; explicit JSON null is rejected by the API.
UPDATE items SET
    text = COALESCE(sqlc.narg('text'), text),
    department = COALESCE(sqlc.narg('department'), department),
    checked_at = CASE WHEN CAST(@set_checked AS INTEGER) THEN
        CASE WHEN CAST(@checked AS INTEGER) THEN COALESCE(checked_at, CAST(@now AS INTEGER)) ELSE NULL END
        ELSE checked_at END
WHERE id = @id AND list_id = @list_id AND deleted_at IS NULL;

-- name: DeleteItem :exec
UPDATE items SET deleted_at = CAST(@now AS INTEGER)
WHERE id = @id AND list_id = @list_id AND deleted_at IS NULL;

-- name: DeleteCheckedItems :exec
UPDATE items SET deleted_at = CAST(@now AS INTEGER)
WHERE list_id = @list_id AND checked_at IS NOT NULL AND deleted_at IS NULL;
