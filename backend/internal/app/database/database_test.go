package database

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "data", "laga.sqlite")

	db, err := Open(ctx, path)

	require.NoError(t, err)

	t.Cleanup(func() {
		err := db.Close()

		assert.NoError(t, err)
	})

	var journalMode string
	err = db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)

	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)
	assert.Equal(t, 1, db.Stats().MaxOpenConnections)

	// Force a replacement connection to verify per-connection settings.
	db.SetMaxIdleConns(0)

	for _, pragma := range []struct {
		name string
		want int
	}{
		{"foreign_keys", 1},
		{"busy_timeout", 5000},
	} {
		var value int
		err := db.QueryRowContext(ctx, "PRAGMA "+pragma.name).Scan(&value)

		require.NoError(t, err)
		assert.Equal(t, pragma.want, value, pragma.name)
	}

	db.SetMaxIdleConns(1)

	_, err = db.ExecContext(ctx, "CREATE TABLE persistence_test (value TEXT NOT NULL)")

	require.NoError(t, err)

	_, err = db.ExecContext(ctx, "INSERT INTO persistence_test VALUES ('saved')")

	require.NoError(t, err)

	err = db.Close()

	require.NoError(t, err)

	reopened, err := Open(ctx, path)

	require.NoError(t, err)

	t.Cleanup(func() {
		err := reopened.Close()

		assert.NoError(t, err)
	})

	var value string
	err = reopened.QueryRowContext(ctx, "SELECT value FROM persistence_test").Scan(&value)

	require.NoError(t, err)
	assert.Equal(t, "saved", value)
}

func TestOpenMemory(t *testing.T) {
	t.Parallel()

	db, err := Open(t.Context(), ":memory:")

	require.NoError(t, err)

	err = db.Close()

	require.NoError(t, err)
}

func TestOpenRequiresPath(t *testing.T) {
	t.Parallel()

	db, err := Open(t.Context(), "")

	require.Error(t, err)
	assert.Nil(t, db)
}
