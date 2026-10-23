package migrations

import (
	"database/sql"
	"testing"

	"github.com/herrnan/handlat/internal/app/database"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestExistingListTimestampMigration(t *testing.T) {
	ctx := t.Context()
	conn, err := database.Open(ctx, ":memory:")

	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, Up(conn))
	require.NoError(t, goose.DownTo(conn, ".", 2))

	_, err = conn.ExecContext(ctx, "INSERT INTO users (id, name) VALUES (?, ?)", []byte("owner"), "Owner")
	require.NoError(t, err)

	const timestamp = "2026-10-23T12:00:00.123456789Z"
	const seconds int64 = 1792756800
	_, err = conn.ExecContext(ctx, "INSERT INTO lists (id, owner_id, name, revision, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)", "list", []byte("owner"), "Groceries", 7, timestamp, timestamp)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, "INSERT INTO items (id, list_id, text, department, created_at, checked_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?)", "deleted", "list", "Milk", "other", timestamp, timestamp, timestamp)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, "INSERT INTO items (id, list_id, text, created_at) VALUES (?, ?, ?, ?)", "active", "list", "Bread", timestamp)
	require.NoError(t, err)

	require.NoError(t, Up(conn))

	var name string
	var created, updated int64
	err = conn.QueryRowContext(ctx, "SELECT name, created_at, updated_at FROM lists WHERE id = ?", "list").Scan(&name, &created, &updated)

	require.NoError(t, err)
	require.Equal(t, "Groceries", name)
	require.Equal(t, seconds, created)
	require.Equal(t, seconds, updated)

	var checked, deleted int64
	err = conn.QueryRowContext(ctx, "SELECT created_at, checked_at, deleted_at FROM items WHERE id = ?", "deleted").Scan(&created, &checked, &deleted)

	require.NoError(t, err)
	require.Equal(t, seconds, created)
	require.Equal(t, seconds, checked)
	require.Equal(t, seconds, deleted)

	var nullChecked, nullDeleted sql.NullInt64
	err = conn.QueryRowContext(ctx, "SELECT checked_at, deleted_at FROM items WHERE id = ?", "active").Scan(&nullChecked, &nullDeleted)

	require.NoError(t, err)
	require.False(t, nullChecked.Valid)
	require.False(t, nullDeleted.Valid)

	var violations int
	err = conn.QueryRowContext(ctx, "SELECT count(*) FROM pragma_foreign_key_check").Scan(&violations)

	require.NoError(t, err)
	require.Zero(t, violations)

	// Rolling back keeps the data, restoring the old timestamp representation.
	// Removed revisions have no meaning anymore and are restored to their default.
	require.NoError(t, goose.DownTo(conn, ".", 2))

	var restoredTimestamp string
	var revision int
	err = conn.QueryRowContext(ctx, "SELECT created_at, revision FROM lists WHERE id = ?", "list").Scan(&restoredTimestamp, &revision)

	require.NoError(t, err)
	require.Equal(t, "2026-10-23T12:00:00Z", restoredTimestamp)
	require.Equal(t, 1, revision)

	require.NoError(t, Up(conn))

	var count int
	err = conn.QueryRowContext(ctx, "SELECT count(*) FROM items WHERE list_id = ?", "list").Scan(&count)

	require.NoError(t, err)
	require.Equal(t, 2, count)
}
