package list

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/herrnan/handlat/internal/app/auth"
	authdb "github.com/herrnan/handlat/internal/app/auth/db"
	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	handler      http.Handler
	conn         *sql.DB
	owner, other string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	conn, err := database.Open(t.Context(), filepath.Join(t.TempDir(), "lists.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, migrations.Up(conn))
	tokens := []string{}
	q := authdb.New(conn)
	for _, name := range []string{"owner", "other"} {
		require.NoError(t, q.InsertUser(t.Context(), authdb.InsertUserParams{ID: []byte(name), Name: name}))
		raw := sha256.Sum256([]byte(name))
		hash := sha256.Sum256(raw[:])
		tokens = append(tokens, base64.RawURLEncoding.EncodeToString(raw[:]))
		require.NoError(t, q.InsertSession(t.Context(), authdb.InsertSessionParams{TokenHash: hash[:], UserID: []byte(name), ExpiresAt: time.Now().Add(time.Hour).Unix()}))
	}
	return fixture{handler: apiHandler(t, conn), conn: conn, owner: tokens[0], other: tokens[1]}
}

func apiHandler(t *testing.T, conn *sql.DB) http.Handler {
	t.Helper()
	authentication, err := auth.New(conn, "localhost", "http://localhost", slog.Default())
	require.NoError(t, err)
	mux := http.NewServeMux()
	New(conn, slog.Default()).Register(mux, authentication.RequireSession)
	return http.NewCrossOriginProtection().Handler(mux)
}

func request(handler http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "laga_session", Value: token})
	}
	req.Header.Set("Content-Type", "application/json")
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, req)
	return result
}

func readList(t *testing.T, response *httptest.ResponseRecorder, status int) List {
	t.Helper()
	require.Equal(t, status, response.Code, response.Body.String())
	var value List
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &value))
	return value
}

func addition(id, text string) string {
	body, err := json.Marshal(map[string]string{"id": id, "text": text})
	if err != nil {
		panic(err)
	}
	return string(body)
}

func TestListLifecycle(t *testing.T) {
	f := newFixture(t)
	list := readList(t, request(f.handler, "POST", "/lists", f.owner, `{"name":"Groceries"}`), 201)
	path := "/lists/" + list.ID
	require.Empty(t, list.Items)
	require.EqualValues(t, 1, list.Revision)
	list = readList(t, request(f.handler, "PATCH", path, f.owner, `{"name":"Weekend"}`), 200)
	require.Equal(t, "Weekend", list.Name)
	id := uuid.NewString()
	list = readList(t, request(f.handler, "POST", path+"/items", f.owner, addition(id, "Milk")), 200)
	require.Len(t, list.Items, 1)
	itemPath := path + "/items/" + id
	list = readList(t, request(f.handler, "PATCH", itemPath, f.owner, `{"checked":true}`), 200)
	checkedAt := list.Items[0].CheckedAt
	require.NotNil(t, checkedAt)
	list = readList(t, request(f.handler, "PATCH", itemPath, f.owner, `{"text":"Oat milk"}`), 200)
	list = readList(t, request(f.handler, "PATCH", itemPath, f.owner, `{"department":"dairy"}`), 200)
	require.Equal(t, "Oat milk", list.Items[0].Text)
	require.Equal(t, "dairy", list.Items[0].Department)
	require.Equal(t, checkedAt, list.Items[0].CheckedAt, "independent edits must preserve checked state")
	list = readList(t, request(f.handler, "PATCH", itemPath, f.owner, `{"checked":false}`), 200)
	require.Nil(t, list.Items[0].CheckedAt)
	list = readList(t, request(f.handler, "DELETE", itemPath, f.owner, ""), 200)
	require.Empty(t, list.Items)
	list = readList(t, request(f.handler, "POST", path+"/items", f.owner, addition(uuid.NewString(), "Apples")), 200)
	require.Len(t, list.Items, 1)
	index := request(f.handler, "GET", "/lists", f.owner, "")
	var summaries []Summary
	require.NoError(t, json.Unmarshal(index.Body.Bytes(), &summaries))
	require.Len(t, summaries, 1)
	require.EqualValues(t, 1, summaries[0].ItemCount, "tombstones must not be counted")
}

func TestBackendResolutionRules(t *testing.T) {
	f := newFixture(t)
	list := readList(t, request(f.handler, "POST", "/lists", f.owner, `{"name":"Rules"}`), 201)
	path := "/lists/" + list.ID
	id := uuid.NewString()
	body := addition(id, "Milk")
	write := func(method, suffix, body string) List {
		t.Helper()
		return readList(t, request(f.handler, method, path+suffix, f.owner, body), 200)
	}
	list = write("POST", "/items", body)
	original := list.Items[0]
	list = write("POST", "/items", addition(id, "A retry must not overwrite the original"))
	require.Equal(t, []Item{original}, list.Items)
	list = write("POST", "/items", addition(uuid.NewString(), "Milk"))
	require.Len(t, list.Items, 2, "identical text with different IDs is a distinct addition")
	itemPath := "/items/" + id
	list = write("PATCH", itemPath, `{"text":"First"}`)
	list = write("PATCH", itemPath, `{"text":"Last"}`)
	find := func(value List, id string) Item {
		t.Helper()
		for _, item := range value.Items {
			if item.ID == id {
				return item
			}
		}
		t.Fatalf("item %s not found", id)
		return Item{}
	}
	require.Equal(t, "Last", find(list, id).Text)
	_ = write("PATCH", "", `{"name":"First name"}`)
	list = write("PATCH", "", `{"name":"Last name"}`)
	require.Equal(t, "Last name", list.Name)

	// Clear operates on current backend state, not a client's earlier snapshot.
	_ = write("PATCH", itemPath, `{"checked":true}`)
	_ = write("PATCH", itemPath, `{"checked":false}`)
	list = write("DELETE", "/checked-items", "")
	require.Len(t, list.Items, 2, "uncheck before clear keeps the item")
	_ = write("PATCH", itemPath, `{"checked":true}`)
	list = write("DELETE", "/checked-items", "")
	require.Len(t, list.Items, 1)
	_ = write("PATCH", itemPath, `{"checked":false,"text":"Too late"}`)
	list = write("POST", "/items", body)
	require.Len(t, list.Items, 1, "neither an edit nor a retried addition can resurrect a cleared item")
	list = write("DELETE", itemPath, "")
	require.Len(t, list.Items, 1, "repeated deletion is harmless")

	// Editing before deletion also leaves the item deleted.
	remaining := list.Items[0].ID
	_ = write("PATCH", "/items/"+remaining, `{"department":"dairy"}`)
	_ = write("DELETE", "/items/"+remaining, "")
	_ = write("PATCH", "/items/"+remaining, `{"text":"Late edit"}`)
	list = write("POST", "/items", addition(remaining, "Milk"))
	require.Empty(t, list.Items)
}

func TestEveryOperationIsPrivate(t *testing.T) {
	f := newFixture(t)
	list := readList(t, request(f.handler, "POST", "/lists", f.owner, `{"name":"Private"}`), 201)
	path := "/lists/" + list.ID
	id := uuid.NewString()
	list = readList(t, request(f.handler, "POST", path+"/items", f.owner, addition(id, "Milk")), 200)
	itemPath := path + "/items/" + id
	for _, op := range []struct{ method, path, body string }{
		{"GET", path, ""},
		{"PATCH", path, `{"name":"Changed"}`},
		{"POST", path + "/items", addition(uuid.NewString(), "Bread")},
		{"PATCH", itemPath, `{"text":"Changed"}`},
		{"DELETE", itemPath, ""},
		{"DELETE", path + "/checked-items", ""},
	} {
		t.Run(op.method+op.path, func(t *testing.T) {
			require.Equal(t, 401, request(f.handler, op.method, op.path, "", op.body).Code)
			require.Equal(t, 404, request(f.handler, op.method, op.path, f.other, op.body).Code)
		})
	}
	require.Equal(t, "[]\n", request(f.handler, "GET", "/lists", f.other, "").Body.String())
	require.Equal(t, 401, request(f.handler, "POST", "/lists", "", `{"name":"New"}`).Code)
	own := readList(t, request(f.handler, "POST", "/lists", f.other, `{"name":"Other"}`), 201)
	// Missing/deleted item commands are no-ops within an accessible list, never
	// permission to edit an item in another list, even with the same item ID.
	result := readList(t, request(f.handler, "PATCH", "/lists/"+own.ID+"/items/"+id, f.other, `{"text":"Attack"}`), 200)
	require.Empty(t, result.Items)
	original := readList(t, request(f.handler, "GET", path, f.owner, ""), 200)
	require.Equal(t, list, original)
}

func TestValidationAndCrossOriginProtection(t *testing.T) {
	f := newFixture(t)
	for _, body := range []string{`{}`, `{"name":" "}`, `{"name":"OK","owner_id":"other"}`, `{"name":"OK"} {}`, `null`} {
		require.Equal(t, 400, request(f.handler, "POST", "/lists", f.owner, body).Code)
	}
	list := readList(t, request(f.handler, "POST", "/lists", f.owner, `{"name":"Private"}`), 201)
	path := "/lists/" + list.ID
	for _, body := range []string{`{"text":"Milk"}`, `{"id":"bad","text":"Milk"}`, `{"id":"00000000-0000-0000-0000-000000000000","text":"Milk"}`, addition(uuid.NewString(), " ")} {
		require.Equal(t, 400, request(f.handler, "POST", path+"/items", f.owner, body).Code)
	}
	for _, body := range []string{`{}`, `null`, `{"text":null}`, `{"department":""}`, `{"text":" "}`, `{"checked":null}`, `{"checked":"true"}`, `{"checkedAt":"2026-10-23T12:00:00Z"}`} {
		require.Equal(t, 400, request(f.handler, "PATCH", path+"/items/missing", f.owner, body).Code)
	}
	current := readList(t, request(f.handler, "GET", path, f.owner, ""), 200)
	require.Equal(t, list.Revision, current.Revision, "invalid commands do not advance sequence")
	req := httptest.NewRequest("POST", "/lists", strings.NewReader(`{"name":"CSRF"}`))
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	res := httptest.NewRecorder()
	f.handler.ServeHTTP(res, req)
	require.Equal(t, 403, res.Code)
}
