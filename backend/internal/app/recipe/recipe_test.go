package recipe_test

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

	"github.com/herrnan/handlat/internal/app/auth"
	authdb "github.com/herrnan/handlat/internal/app/auth/db"
	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/herrnan/handlat/internal/app/recipe"
)

func requireOK(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
}

func createSession(t *testing.T, conn *sql.DB, user string) string {
	t.Helper()

	q := authdb.New(conn)
	requireOK(t, q.InsertUser(t.Context(), authdb.InsertUserParams{ID: []byte(user), Name: user}))

	raw := sha256.Sum256([]byte(user))
	hash := sha256.Sum256(raw[:])
	requireOK(t, q.InsertSession(t.Context(), authdb.InsertSessionParams{
		TokenHash: hash[:], UserID: []byte(user), ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}))

	return base64.RawURLEncoding.EncodeToString(raw[:])
}

func TestPrivateRecipeCollection(t *testing.T) {
	conn, err := database.Open(t.Context(), filepath.Join(t.TempDir(), "recipes.sqlite"))
	requireOK(t, err)
	t.Cleanup(func() { requireOK(t, conn.Close()) })
	requireOK(t, migrations.Up(conn))

	owner := createSession(t, conn, "owner")
	guest := createSession(t, conn, "guest")
	authentication, err := auth.New(conn, "localhost", "http://localhost", slog.Default())
	requireOK(t, err)

	mux := http.NewServeMux()
	recipe.New(conn, slog.Default()).Register(mux, authentication.RequireSession)

	request := func(session, method, path, body string, status int) string {
		t.Helper()

		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if session != "" {
			req.AddCookie(&http.Cookie{Name: "laga_session", Value: session})
		}

		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)

		if res.Code != status {
			t.Fatalf(
				"%s %s: status = %d, want %d, body = %s",
				method,
				path,
				res.Code,
				status,
				res.Body,
			)
		}

		return strings.TrimSpace(res.Body.String())
	}
	collectionCount := func(want int) {
		t.Helper()

		var count int

		err := conn.QueryRowContext(t.Context(), "SELECT count(*) FROM recipe_collections").
			Scan(&count)
		requireOK(t, err)

		if count != want {
			t.Fatalf("collections = %d, want %d", count, want)
		}
	}

	const (
		root   = "/recipes"
		enable = "/recipes/enable"
		id     = "68a96b78-b9c6-4fee-a942-8f5184d78b50"
		path   = root + "/" + id
		draft  = `{"name":"Salsa","portions":4,"ingredients":[{"name":"Tomatoes","amount":"400","unit":"g"},{"name":"Cilantro"}]}`
	)

	body := `{"id":"` + id + `",` + draft[1:]

	request("", http.MethodGet, root, "", http.StatusUnauthorized)

	disabled := request(owner, http.MethodGet, root, "", http.StatusOK)
	if disabled != `{"enabled":false,"recipes":[]}` {
		t.Fatalf("unexpected disabled overview: %s", disabled)
	}

	request(owner, http.MethodPost, root, body, http.StatusForbidden)
	request(owner, http.MethodPost, enable, "", http.StatusNoContent)

	invalid := strings.Replace(body, `"portions":4`, `"portions":0`, 1)
	request(owner, http.MethodPost, root, invalid, http.StatusBadRequest)

	for _, amount := range []string{"0", "-1", "1e3", strings.Repeat("1", 101)} {
		invalidAmount := strings.Replace(body, `"amount":"400"`, `"amount":"`+amount+`"`, 1)
		request(owner, http.MethodPost, root, invalidAmount, http.StatusBadRequest)
	}

	tooManyIngredients := `{"id":"` + id + `","name":"Salsa","portions":4,"ingredients":[` +
		strings.Repeat(`{"name":"Lime"},`, 100) + `{"name":"Salt"}]}`
	request(owner, http.MethodPost, root, tooManyIngredients, http.StatusBadRequest)

	collectionCount(0)

	first := request(owner, http.MethodPost, root, body, http.StatusCreated)

	retry := strings.Replace(body, `"amount":"400"`, `"amount":"500"`, 1)

	second := request(owner, http.MethodPost, root, retry, http.StatusCreated)
	if first != second {
		t.Fatalf("retry changed first recipe: %s vs %s", first, second)
	}

	var saved struct {
		Portions    int
		Ingredients []struct {
			Name   string
			Amount string
			Unit   string
		}
	}
	requireOK(t, json.Unmarshal([]byte(first), &saved))

	if saved.Portions != 4 || len(saved.Ingredients) != 2 ||
		saved.Ingredients[0].Name != "Tomatoes" || saved.Ingredients[0].Amount != "400" ||
		saved.Ingredients[0].Unit != "g" || saved.Ingredients[1].Name != "Cilantro" ||
		saved.Ingredients[1].Amount != "" {
		t.Fatalf("unexpected saved recipe: %+v", saved)
	}

	another := strings.Replace(body, id, "53ee1a5f-2fe0-4640-b15e-cee8f340683c", 1)
	request(owner, http.MethodPost, root, another, http.StatusCreated)
	collectionCount(1)

	request(guest, http.MethodPost, enable, "", http.StatusNoContent)

	if index := request(
		guest,
		http.MethodGet,
		root,
		"",
		http.StatusOK,
	); index != `{"enabled":true,"recipes":[]}` {
		t.Fatalf("guest saw recipes in another collection: %s", index)
	}

	request(guest, http.MethodGet, path, "", http.StatusNotFound)
	request(guest, http.MethodPut, path, draft, http.StatusNotFound)
	request(guest, http.MethodDelete, path, "", http.StatusNotFound)
	request(guest, http.MethodPost, root, body, http.StatusNotFound)
	collectionCount(1)

	const replacement = `{"name":"Tomato salsa","portions":2,"ingredients":[{"name":"Cilantro","amount":"1,5","unit":"msk"}]}`

	request(owner, http.MethodPut, path, replacement, http.StatusOK)

	updated := request(owner, http.MethodGet, path, "", http.StatusOK)
	want := `{"id":"` + id + `",` + strings.Replace(replacement[1:], `"1,5"`, `"1.5"`, 1)

	if updated != want {
		t.Fatalf("updated recipe = %s, want %s", updated, want)
	}

	request(owner, http.MethodDelete, path, "", http.StatusNoContent)
	request(owner, http.MethodGet, path, "", http.StatusNotFound)
	collectionCount(1)
}
