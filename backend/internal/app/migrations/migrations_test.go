package migrations

import (
	"path/filepath"
	"testing"

	"github.com/herrnan/laga/internal/app/database"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpAndDown(t *testing.T) {
	ctx := t.Context()

	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "laga.sqlite"))

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

	for _, table := range []string{"users", "credentials", "registration_attempts", "login_attempts", "sessions"} {
		var exists bool
		err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type = 'table' AND name = ?)", table).
			Scan(&exists)

		require.NoError(t, err)
		assert.True(t, exists, "missing migrated table %s", table)
	}

	err = goose.Down(db, ".")

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
