package migrations_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestPackageUnitMigration(t *testing.T) {
	ctx := t.Context()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "package-unit.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, migrations.Up(db))
	require.NoError(t, goose.DownTo(db, ".", 5))

	_, err = db.ExecContext(
		ctx,
		"INSERT INTO lists (id, name, created_at, updated_at) VALUES ('list', 'Groceries', 1, 1)",
	)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO items (id, list_id, text, created_at, amount, unit) VALUES
			('pasta', 'list', 'Pasta', 1, '1.5', 'förp'),
			('salt', 'list', 'Salt', 1, '2', 'tsk'),
			('milk', 'list', 'Milk', 1, NULL, NULL)
	`)
	require.NoError(t, err)

	checkItem := func(id, expectedAmount, expectedUnit string) {
		t.Helper()

		var amount, unit sql.NullString
		require.NoError(
			t,
			db.QueryRowContext(ctx, "SELECT amount, unit FROM items WHERE id = ?", id).
				Scan(&amount, &unit),
		)
		require.Equal(
			t,
			sql.NullString{String: expectedAmount, Valid: expectedAmount != ""},
			amount,
		)
		require.Equal(t, sql.NullString{String: expectedUnit, Valid: expectedUnit != ""}, unit)
	}

	require.NoError(t, goose.UpTo(db, ".", 6))
	checkItem("pasta", "1.5", "pkt")
	checkItem("salt", "2", "tsk")
	checkItem("milk", "", "")

	require.NoError(t, goose.DownTo(db, ".", 5))
	checkItem("pasta", "1.5", "förp")
	checkItem("salt", "2", "tsk")
	checkItem("milk", "", "")
}
