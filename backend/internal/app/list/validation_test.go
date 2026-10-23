package list

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func jsonBody(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

func TestLengthLimitsBeforeDatabaseWrites(t *testing.T) {
	for _, character := range []string{"a", "🥕"} {
		t.Run(character, func(t *testing.T) {
			f := newFixture(t)
			name := strings.Repeat(character, 200)
			tooLongName := jsonBody(t, map[string]string{"name": name + character})

			response := request(f.handler, "POST", "/lists", f.owner, tooLongName)
			require.Equal(t, 400, response.Code)
			require.Contains(t, response.Body.String(), "1–200 characters")
			require.Equal(t, "[]\n", request(f.handler, "GET", "/lists", f.owner, "").Body.String())

			list := readList(t, request(f.handler, "POST", "/lists", f.owner, jsonBody(t, map[string]string{"name": name})), 201)
			path := "/lists/" + list.ID
			require.Equal(t, name, list.Name)

			response = request(f.handler, "PATCH", path, f.owner, tooLongName)
			require.Equal(t, 400, response.Code)
			require.Contains(t, response.Body.String(), "1–200 characters")

			text := strings.Repeat(character, 2000)
			response = request(f.handler, "POST", path+"/items", f.owner, addition(uuid.NewString(), text+character))
			require.Equal(t, 400, response.Code)
			require.Contains(t, response.Body.String(), "1–2000 characters")
			require.Equal(t, list, readList(t, request(f.handler, "GET", path, f.owner, ""), 200))

			list = readList(t, request(f.handler, "POST", path+"/items", f.owner, addition(uuid.NewString(), text)), 200)
			require.Equal(t, text, list.Items[0].Text)

			itemPath := path + "/items/" + list.Items[0].ID
			response = request(f.handler, "PATCH", itemPath, f.owner, jsonBody(t, map[string]string{"text": text + character}))
			require.Equal(t, 400, response.Code)
			require.Contains(t, response.Body.String(), "1–2000 characters")
			require.Equal(t, list, readList(t, request(f.handler, "GET", path, f.owner, ""), 200))

			// Renaming and editing also accept the exact boundary, not just creation.
			require.Equal(t, 200, request(f.handler, "PATCH", path, f.owner, jsonBody(t, map[string]string{"name": name})).Code)
			require.Equal(t, 200, request(f.handler, "PATCH", itemPath, f.owner, jsonBody(t, map[string]string{"text": text})).Code)
		})
	}
}

func TestTimestampStorageAndAPIRepresentation(t *testing.T) {
	f := newFixture(t)
	response := request(f.handler, "POST", "/lists", f.owner, `{"name":"Timestamps"}`)
	require.Empty(t, response.Header().Get("ETag"))
	require.NotContains(t, response.Body.String(), `"revision"`)
	list := readList(t, response, 201)
	path := "/lists/" + list.ID
	list = readList(t, request(f.handler, "POST", path+"/items", f.owner, addition(uuid.NewString(), "Milk")), 200)
	itemPath := path + "/items/" + list.Items[0].ID
	list = readList(t, request(f.handler, "PATCH", itemPath, f.owner, `{"checked":true}`), 200)

	created, err := time.Parse(time.RFC3339, list.Items[0].CreatedAt)
	require.NoError(t, err)
	checked, err := time.Parse(time.RFC3339, *list.Items[0].CheckedAt)
	require.NoError(t, err)

	var createdAt, checkedAt int64
	var createdType, checkedType string
	err = f.conn.QueryRowContext(
		t.Context(),
		"SELECT created_at, checked_at, typeof(created_at), typeof(checked_at) FROM items WHERE list_id = ? AND id = ?",
		list.ID, list.Items[0].ID,
	).Scan(&createdAt, &checkedAt, &createdType, &checkedType)

	require.NoError(t, err)
	require.Equal(t, created.Unix(), createdAt)
	require.Equal(t, checked.Unix(), checkedAt)
	require.Equal(t, "integer", createdType)
	require.Equal(t, "integer", checkedType)

	require.Equal(t, 200, request(f.handler, "DELETE", itemPath, f.owner, "").Code)
	var deletedAt int64
	var deletedType string
	err = f.conn.QueryRowContext(t.Context(), "SELECT deleted_at, typeof(deleted_at) FROM items WHERE list_id = ? AND id = ?", list.ID, list.Items[0].ID).Scan(&deletedAt, &deletedType)

	require.NoError(t, err)
	require.Positive(t, deletedAt)
	require.Equal(t, "integer", deletedType)
}
