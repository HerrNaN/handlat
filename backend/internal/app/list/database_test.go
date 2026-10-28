package list_test

import (
	"path/filepath"
	"testing"

	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/list/db"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/stretchr/testify/require"
)

func TestListSchemaConstraints(t *testing.T) {
	ctx := t.Context()
	conn, err := database.Open(ctx, filepath.Join(t.TempDir(), "lists.sqlite"))

	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, migrations.Up(conn))

	_, err = conn.ExecContext(
		ctx,
		"INSERT INTO users (id, name) VALUES (?, ?)",
		[]byte("owner"),
		"Owner",
	)

	require.NoError(t, err)

	q := db.New(conn)
	params := db.CreateListParams{
		ID:      "list",
		OwnerID: []byte("owner"),
		Name:    "Groceries",
		Now:     1792756800,
	}

	require.NoError(t, q.CreateList(ctx, params))

	t.Run("foreign keys reject orphan lists", func(t *testing.T) {
		orphan := params
		orphan.ID = "orphan"
		orphan.OwnerID = []byte("missing")

		require.Error(t, q.CreateList(ctx, orphan))
	})

	t.Run("deleting a list cascades to its items", func(t *testing.T) {
		err := q.AddItem(ctx, db.AddItemParams{
			ID:         "item",
			ListID:     "list",
			Text:       "Milk",
			Department: "other",
			CreatedAt:  params.Now,
		})

		require.NoError(t, err)

		_, err = conn.ExecContext(ctx, "DELETE FROM lists WHERE id = ?", "list")

		require.NoError(t, err)

		var count int

		err = conn.QueryRowContext(ctx, "SELECT count(*) FROM items WHERE list_id = ?", "list").
			Scan(&count)

		require.NoError(t, err)
		require.Zero(t, count)
	})
}
