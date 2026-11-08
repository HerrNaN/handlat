package list_test

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	listapi "github.com/herrnan/handlat/internal/app/list"
	listdb "github.com/herrnan/handlat/internal/app/list/db"
	"github.com/stretchr/testify/require"
)

func openStream(t *testing.T, server *httptest.Server, token, id string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		server.URL+"/lists/"+id+"/events",
		nil,
	)
	require.NoError(t, err)

	if token != "" {
		req.AddCookie(&http.Cookie{Name: "laga_session", Value: token})
	}

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, response.Body.Close()) })

	return response
}

func readSnapshot(t *testing.T, reader *bufio.Reader) listapi.List {
	t.Helper()

	line, err := reader.ReadString('\n')
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(line, "data: "), line)

	var result listapi.List
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &result))
	line, err = reader.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "\n", line)

	return result
}

func TestStreamSnapshotsAndReconnect(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, listdb.New(f.conn).CreateList(t.Context(), listdb.CreateListParams{
		ID: "list", OwnerID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))

	server := httptest.NewServer(f.handler)
	defer server.Close()

	response := openStream(t, server, f.owner, "list")
	defer func() { require.NoError(t, response.Body.Close()) }()

	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))
	reader := bufio.NewReader(response.Body)
	initial := readSnapshot(t, reader)
	require.Equal(t, "Groceries", initial.Name)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/lists/list",
		strings.NewReader(`{"name":"Weekend"}`),
	)
	request.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})

	saved := httptest.NewRecorder()
	f.handler.ServeHTTP(saved, request)
	require.Equal(t, http.StatusOK, saved.Code)
	update := readSnapshot(t, reader)
	require.Equal(t, "Weekend", update.Name)

	var httpSnapshot listapi.List
	require.NoError(t, json.Unmarshal(saved.Body.Bytes(), &httpSnapshot))
	require.Equal(t, httpSnapshot, update)
	require.NoError(t, response.Body.Close())

	reconnected := openStream(t, server, f.owner, "list")
	defer func() { require.NoError(t, reconnected.Body.Close()) }()

	require.Equal(t, update, readSnapshot(t, bufio.NewReader(reconnected.Body)))

	request = httptest.NewRequest(http.MethodDelete, "/lists/list", nil)
	request.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})

	saved = httptest.NewRecorder()
	f.handler.ServeHTTP(saved, request)
	require.Equal(t, http.StatusNoContent, saved.Code)

	reader = bufio.NewReader(reconnected.Body)
	event, err := reader.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "event: deleted\n", event)
}

func TestStreamAuthorization(t *testing.T) {
	f := newFixture(t)
	_, err := f.conn.ExecContext(
		t.Context(),
		"INSERT INTO users (id, name) VALUES (?, ?)",
		[]byte("other"),
		"Other",
	)
	require.NoError(t, err)
	require.NoError(t, listdb.New(f.conn).CreateList(t.Context(), listdb.CreateListParams{
		ID: "private", OwnerID: []byte("other"), Name: "Private", Now: time.Now().Unix(),
	}))

	server := httptest.NewServer(f.handler)
	defer server.Close()

	for _, tc := range []struct {
		name, token, id string
		status          int
	}{
		{"no session", "", "private", http.StatusUnauthorized},
		{"other owner", f.owner, "private", http.StatusNotFound},
		{"missing", f.owner, "missing", http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := openStream(t, server, tc.token, tc.id)
			require.Equal(t, tc.status, response.StatusCode)
		})
	}
}
