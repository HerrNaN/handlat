-- name: CreateList :exec
INSERT INTO lists (id, name, created_at, updated_at)
VALUES (@id, @name, @now, @now);

-- name: GetList :one
SELECT * FROM lists WHERE id = @id;

-- name: ListLists :many
SELECT lists.id, lists.name, count(items.id) AS item_count,
    (SELECT count(*) FROM list_members WHERE list_id = lists.id) AS member_count,
    EXISTS (SELECT 1 FROM default_shopping_lists
        WHERE default_shopping_lists.list_id = lists.id AND default_shopping_lists.user_id = @user_id) AS is_default
FROM lists LEFT JOIN items ON items.list_id = lists.id AND items.deleted_at IS NULL
WHERE EXISTS (SELECT 1 FROM list_members WHERE list_members.list_id = lists.id AND list_members.user_id = @user_id)
GROUP BY lists.id ORDER BY lists.created_at, lists.id;

-- name: SetDefaultShoppingList :one
INSERT INTO default_shopping_lists (list_id, user_id)
SELECT list_members.list_id, list_members.user_id FROM list_members
WHERE list_members.list_id = @list_id AND list_members.user_id = @user_id
ON CONFLICT (user_id) DO UPDATE SET list_id = excluded.list_id
RETURNING default_shopping_lists.list_id;

-- name: EnsureDefaultShoppingList :exec
INSERT INTO default_shopping_lists (user_id, list_id)
SELECT list_members.user_id, lists.id FROM lists
JOIN list_members ON list_members.list_id = lists.id
WHERE list_members.user_id = @user_id
ORDER BY lists.created_at, lists.id
LIMIT 1
ON CONFLICT (user_id) DO NOTHING;

-- name: TouchList :exec
UPDATE lists SET updated_at = @now
WHERE id = @id;

-- name: RenameList :exec
UPDATE lists SET name = @name WHERE id = @id;

-- name: DeleteEmptyList :exec
DELETE FROM lists WHERE id = @id AND NOT EXISTS (SELECT 1 FROM list_members WHERE list_id = lists.id);

-- name: ListItems :many
SELECT * FROM items WHERE list_id = @list_id AND deleted_at IS NULL ORDER BY created_at, id;

-- name: ListMergeCandidates :many
SELECT candidate.id, candidate.text, candidate.amount, candidate.unit
FROM items AS candidate
WHERE candidate.list_id = @list_id
    AND candidate.deleted_at IS NULL AND candidate.checked_at IS NULL AND candidate.amount IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM items AS prior
        WHERE prior.list_id = @list_id AND prior.id = @incoming_id
    )
ORDER BY candidate.created_at, candidate.id;

-- name: AddItem :exec
INSERT INTO items (id, list_id, text, department, created_at, amount, unit, deleted_at)
VALUES (@id, @list_id, @text, @department, @created_at, sqlc.narg('amount'), sqlc.narg('unit'), sqlc.narg('deleted_at'))
ON CONFLICT (list_id, id) DO NOTHING;

-- name: UpdateItem :exec
-- narg makes omitted PATCH fields nullable parameters despite NOT NULL columns.
-- COALESCE preserves text and department; explicit null is rejected for those fields.
-- Separate presence flags let amount and unit be explicitly cleared with null.
UPDATE items SET
    text = COALESCE(sqlc.narg('text'), text),
    department = COALESCE(sqlc.narg('department'), department),
    amount = CASE WHEN CAST(@set_amount AS INTEGER) THEN sqlc.narg('amount') ELSE amount END,
    unit = CASE WHEN CAST(@set_unit AS INTEGER) THEN sqlc.narg('unit') ELSE unit END,
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

-- name: GetMemberList :one
SELECT lists.* FROM lists
JOIN list_members ON list_members.list_id = lists.id
WHERE lists.id = @id AND list_members.user_id = @user_id;

-- name: AddMember :exec
INSERT INTO list_members (list_id, user_id) VALUES (@list_id, @user_id)
ON CONFLICT (list_id, user_id) DO NOTHING;

-- name: RemoveMember :exec
DELETE FROM list_members WHERE list_id = @list_id AND user_id = @user_id;

-- name: CountMembers :one
SELECT count(*) FROM list_members WHERE list_id = @list_id;

-- name: CreateInvitation :exec
INSERT INTO list_invitations (token_hash, list_id, expires_at)
VALUES (@token_hash, @list_id, @expires_at);

-- name: GetInvitation :one
SELECT lists.id, lists.name FROM list_invitations
JOIN lists ON lists.id = list_invitations.list_id
WHERE token_hash = @token_hash AND expires_at > @now;

-- name: ConsumeInvitation :one
DELETE FROM list_invitations WHERE token_hash = @token_hash AND expires_at > @now
RETURNING list_id;

-- name: DeleteExpiredInvitations :exec
DELETE FROM list_invitations WHERE expires_at <= @now;
