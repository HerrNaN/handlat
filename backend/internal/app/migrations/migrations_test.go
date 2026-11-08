package migrations

import (
	"path/filepath"
	"testing"

	"github.com/herrnan/handlat/internal/app/database"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpAndDown(t *testing.T) {
	ctx := t.Context()

	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "handlat.sqlite"))

	require.NoError(t, err)

	t.Cleanup(func() {
		err := db.Close()

		assert.NoError(t, err)
	})

	err = Up(db)

	require.NoError(t, err)

	err = Up(db)

	require.NoError(t, err)

	// Migrations must leave the application connection open.
	err = db.PingContext(ctx)

	require.NoError(t, err)

	for _, table := range []string{"users", "credentials", "registration_attempts", "login_attempts", "sessions", "lists", "items"} {
		var exists bool

		err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type = 'table' AND name = ?)", table).
			Scan(&exists)

		require.NoError(t, err)
		assert.True(t, exists, "missing migrated table %s", table)
	}

	err = goose.DownTo(db, ".", 0)

	require.NoError(t, err)

	var tables int

	err = db.QueryRowContext(
		ctx,
		"SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name NOT IN ('goose_db_version', 'sqlite_sequence')",
	).Scan(&tables)

	require.NoError(t, err)
	assert.Zero(t, tables)

	err = Up(db)

	require.NoError(t, err)
}

func TestSharingPreservesExistingLists(t *testing.T) {
	ctx := t.Context()
	conn, err := database.Open(ctx, filepath.Join(t.TempDir(), "existing.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	goose.SetBaseFS(files)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(conn, ".", 2))
	_, err = conn.ExecContext(
		ctx,
		"INSERT INTO users (id, name) VALUES (?, ?)",
		[]byte("existing"),
		"Existing",
	)
	require.NoError(t, err)
	_, err = conn.ExecContext(
		ctx,
		"INSERT INTO lists (id, owner_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		"groceries",
		[]byte("existing"),
		"Groceries",
		1,
		1,
	)
	require.NoError(t, err)
	_, err = conn.ExecContext(
		ctx,
		"INSERT INTO items (id, list_id, text, department, created_at) VALUES (?, ?, ?, ?, ?)",
		"milk",
		"groceries",
		"Milk",
		"other",
		1,
	)
	require.NoError(t, err)
	require.NoError(t, Up(conn))

	var member []byte
	require.NoError(
		t,
		conn.QueryRowContext(ctx, "SELECT user_id FROM list_members WHERE list_id = ?", "groceries").
			Scan(&member),
	)
	require.Equal(t, []byte("existing"), member)

	var text string
	require.NoError(
		t,
		conn.QueryRowContext(ctx, "SELECT text FROM items WHERE list_id = ?", "groceries").
			Scan(&text),
	)
	require.Equal(t, "Milk", text)
	require.NoError(t, goose.DownTo(conn, ".", 2))

	var owner []byte
	require.NoError(
		t,
		conn.QueryRowContext(ctx, "SELECT owner_id FROM lists WHERE id = ?", "groceries").
			Scan(&owner),
	)
	require.Equal(t, member, owner)
	require.NoError(
		t,
		conn.QueryRowContext(ctx, "SELECT text FROM items WHERE list_id = ?", "groceries").
			Scan(&text),
	)
	require.Equal(t, "Milk", text)
	require.NoError(t, Up(conn))
}
