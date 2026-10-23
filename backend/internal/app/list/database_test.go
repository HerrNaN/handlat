package list

import (
	"path/filepath"
	"testing"

	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/list/db"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/stretchr/testify/require"
)

func TestListQueries(t *testing.T) {
	ctx := t.Context()
	conn, err := database.Open(ctx, filepath.Join(t.TempDir(), "lists.sqlite"))

	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, migrations.Up(conn))

	_, err = conn.ExecContext(ctx, "INSERT INTO users (id, name) VALUES (?, ?)", []byte("owner"), "Owner")

	require.NoError(t, err)

	q := db.New(conn)
	params := db.CreateListParams{
		ID:      "list",
		OwnerID: []byte("owner"),
		Name:    "Groceries",
		Now:     1792756800,
	}

	require.NoError(t, q.CreateList(ctx, params))

	t.Run("summaries count items and filter by owner", func(t *testing.T) {
		err := q.AddItem(ctx, db.AddItemParams{
			ID:         "item",
			ListID:     "list",
			Text:       "Milk",
			Department: "other",
			CreatedAt:  params.Now,
		})

		require.NoError(t, err)

		lists, err := q.ListLists(ctx, params.OwnerID)

		require.NoError(t, err)
		require.Len(t, lists, 1)
		require.EqualValues(t, 1, lists[0].ItemCount)

		other, err := q.ListLists(ctx, []byte("other"))

		require.NoError(t, err)
		require.Empty(t, other)
	})

	t.Run("timestamps are stored as Unix seconds", func(t *testing.T) {
		err := q.TouchList(ctx, db.TouchListParams{ID: "list", Now: params.Now + 1})

		require.NoError(t, err)

		value, err := q.GetList(ctx, "list")

		require.NoError(t, err)
		require.Equal(t, params.Now, value.CreatedAt)
		require.Equal(t, params.Now+1, value.UpdatedAt)

		var storageType string
		err = conn.QueryRowContext(ctx, "SELECT typeof(updated_at) FROM lists WHERE id = ?", "list").Scan(&storageType)

		require.NoError(t, err)
		require.Equal(t, "integer", storageType)
	})

	t.Run("foreign keys reject orphan lists and cascade item deletion", func(t *testing.T) {
		orphan := params
		orphan.ID = "orphan"
		orphan.OwnerID = []byte("missing")

		require.Error(t, q.CreateList(ctx, orphan))

		_, err := conn.ExecContext(ctx, "DELETE FROM lists WHERE id = ?", "list")

		require.NoError(t, err)

		items, err := q.ListItems(ctx, "list")

		require.NoError(t, err)
		require.Empty(t, items)
	})
}
