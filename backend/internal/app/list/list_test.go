package list_test

import (
	"context"
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
	listapi "github.com/herrnan/handlat/internal/app/list"
	listdb "github.com/herrnan/handlat/internal/app/list/db"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	handler http.Handler
	conn    *sql.DB
	owner   string
}

type seedListParams struct {
	ID     string
	UserID []byte
	Name   string
	Now    int64
}

func seedList(ctx context.Context, q *listdb.Queries, params seedListParams) error {
	if err := q.CreateList(
		ctx,
		listdb.CreateListParams{ID: params.ID, Name: params.Name, Now: params.Now},
	); err != nil {
		return err
	}

	return q.AddMember(ctx, listdb.AddMemberParams{ListID: params.ID, UserID: params.UserID})
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	conn, err := database.Open(t.Context(), filepath.Join(t.TempDir(), "lists.sqlite"))
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, migrations.Up(conn))

	q := authdb.New(conn)
	require.NoError(
		t,
		q.InsertUser(t.Context(), authdb.InsertUserParams{ID: []byte("owner"), Name: "owner"}),
	)

	raw := sha256.Sum256([]byte("owner"))
	hash := sha256.Sum256(raw[:])
	token := base64.RawURLEncoding.EncodeToString(raw[:])

	require.NoError(t, q.InsertSession(t.Context(), authdb.InsertSessionParams{
		TokenHash: hash[:], UserID: []byte("owner"), ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}))

	return fixture{handler: apiHandler(t, conn), conn: conn, owner: token}
}

func apiHandler(t *testing.T, conn *sql.DB) http.Handler {
	t.Helper()

	authentication, err := auth.New(conn, "localhost", "http://localhost", slog.Default())
	require.NoError(t, err)

	mux := http.NewServeMux()
	listapi.New(conn, slog.Default()).Register(mux, authentication.RequireSession)

	return mux
}

func TestCreateList(t *testing.T) {
	f := newFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/lists", strings.NewReader(`{"name":"Groceries"}`))
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})
	req.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	f.handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusCreated, response.Code)

	var list listapi.List
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &list))

	_, err := uuid.Parse(list.ID)
	require.NoError(t, err)
	require.Equal(t, "Groceries", list.Name)
	require.Empty(t, list.Items)
	require.Equal(t, "/api/lists/"+list.ID, response.Header().Get("Location"))
}

func TestGetList(t *testing.T) {
	f := newFixture(t)
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))
	require.NoError(t, q.AddItem(t.Context(), listdb.AddItemParams{
		ID: "milk", ListID: "list", Text: "Milk", Department: "other", CreatedAt: time.Now().Unix(),
	}))

	req := httptest.NewRequest(http.MethodGet, "/lists/list", nil)
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})

	response := httptest.NewRecorder()

	f.handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)

	var list listapi.List
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &list))
	require.Equal(t, "list", list.ID)
	require.Equal(t, "Groceries", list.Name)
	require.Len(t, list.Items, 1)
	require.Equal(t, "milk", list.Items[0].ID)
	require.Equal(t, "Milk", list.Items[0].Text)
}

func TestRenameList(t *testing.T) {
	f := newFixture(t)
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))
	require.NoError(t, q.AddItem(t.Context(), listdb.AddItemParams{
		ID: "milk", ListID: "list", Text: "Milk", Department: "other", CreatedAt: time.Now().Unix(),
	}))

	req := httptest.NewRequest(
		http.MethodPatch,
		"/lists/list",
		strings.NewReader(`{"name":"Weekend"}`),
	)
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})
	req.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	f.handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)

	var list listapi.List
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &list))
	require.Equal(t, "list", list.ID)
	require.Equal(t, "Weekend", list.Name)
	require.Len(t, list.Items, 1)
	require.Equal(t, "milk", list.Items[0].ID)
	require.Equal(t, "Milk", list.Items[0].Text)
}

func TestDeleteList(t *testing.T) {
	for _, tc := range []struct {
		name          string
		owner         string
		id            string
		authenticated bool
		status        int
	}{
		{"owner", "owner", "list", true, http.StatusNoContent},
		{"other owner", "other", "list", true, http.StatusNotFound},
		{"missing", "owner", "missing", true, http.StatusNotFound},
		{"unauthenticated", "owner", "list", false, http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			_, err := f.conn.ExecContext(
				t.Context(),
				"INSERT INTO users (id, name) VALUES (?, ?)",
				[]byte("other"),
				"Other",
			)
			require.NoError(t, err)

			q := listdb.New(f.conn)
			require.NoError(t, seedList(t.Context(), q, seedListParams{
				ID: "list", UserID: []byte(tc.owner), Name: "Groceries", Now: time.Now().Unix(),
			}))
			require.NoError(t, q.AddItem(t.Context(), listdb.AddItemParams{
				ID:         "milk",
				ListID:     "list",
				Text:       "Milk",
				Department: "other",
				CreatedAt:  time.Now().Unix(),
			}))

			req := httptest.NewRequest(http.MethodDelete, "/lists/"+tc.id, nil)
			if tc.authenticated {
				req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})
			}

			response := httptest.NewRecorder()
			f.handler.ServeHTTP(response, req)
			require.Equal(t, tc.status, response.Code)

			var count int
			require.NoError(
				t,
				f.conn.QueryRowContext(t.Context(), "SELECT count(*) FROM items WHERE list_id = ?", "list").
					Scan(&count),
			)

			if tc.status == http.StatusNoContent {
				require.Empty(t, response.Body.String())
				require.Zero(t, count)
				_, err := q.GetList(t.Context(), "list")
				require.ErrorIs(t, err, sql.ErrNoRows)
				rows, err := q.ListLists(t.Context(), []byte("owner"))
				require.NoError(t, err)
				require.Empty(t, rows)

				response = httptest.NewRecorder()
				f.handler.ServeHTTP(response, req)
				require.Equal(t, http.StatusNotFound, response.Code)
			} else {
				require.Equal(t, 1, count)
				_, err := q.GetList(t.Context(), "list")
				require.NoError(t, err)
			}
		})
	}
}

func TestAddItem(t *testing.T) {
	f := newFixture(t)
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))

	id := uuid.NewString()
	req := httptest.NewRequest(
		http.MethodPost,
		"/lists/list/items",
		strings.NewReader(`{"id":"`+id+`","text":"Milk"}`),
	)
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})
	req.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	f.handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)

	var list listapi.List
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &list))
	require.Len(t, list.Items, 1)
	require.Equal(t, id, list.Items[0].ID)
	require.Equal(t, "Milk", list.Items[0].Text)
	require.Equal(t, "other", list.Items[0].Department)
	require.NotEmpty(t, list.Items[0].CreatedAt)
	require.Nil(t, list.Items[0].CheckedAt)
}

func TestUpdateItem(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		text       string
		department string
		checked    bool
	}{
		{"text", `{"text":"Oat milk"}`, "Oat milk", "other", true},
		{"department", `{"department":"dairy"}`, "Milk", "dairy", true},
		{"check", `{"checked":true}`, "Milk", "other", true},
		{"uncheck", `{"checked":false}`, "Milk", "other", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			q := listdb.New(f.conn)
			now := time.Now().Unix()
			require.NoError(t, seedList(t.Context(), q, seedListParams{
				ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: now,
			}))
			require.NoError(t, q.AddItem(t.Context(), listdb.AddItemParams{
				ID: "milk", ListID: "list", Text: "Milk", Department: "other", CreatedAt: now,
			}))

			if tc.name != "check" {
				require.NoError(t, q.UpdateItem(t.Context(), listdb.UpdateItemParams{
					ID: "milk", ListID: "list", SetChecked: 1, Checked: 1, Now: now,
				}))
			}

			req := httptest.NewRequest(
				http.MethodPatch,
				"/lists/list/items/milk",
				strings.NewReader(tc.body),
			)
			req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})
			req.Header.Set("Content-Type", "application/json")

			response := httptest.NewRecorder()

			f.handler.ServeHTTP(response, req)
			require.Equal(t, http.StatusOK, response.Code)

			var list listapi.List
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &list))
			require.Len(t, list.Items, 1)
			require.Equal(t, "milk", list.Items[0].ID)
			require.Equal(t, time.Unix(now, 0).UTC(), list.Items[0].CreatedAt)
			require.Equal(t, tc.text, list.Items[0].Text)
			require.Equal(t, tc.department, list.Items[0].Department)

			if tc.checked {
				require.NotNil(t, list.Items[0].CheckedAt)
			} else {
				require.Nil(t, list.Items[0].CheckedAt)
			}

			if tc.name == "text" || tc.name == "department" {
				require.Equal(
					t,
					time.Unix(now, 0).UTC(),
					*list.Items[0].CheckedAt,
				)
			}
		})
	}
}

func TestDeleteItem(t *testing.T) {
	f := newFixture(t)
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))
	require.NoError(t, q.AddItem(t.Context(), listdb.AddItemParams{
		ID: "milk", ListID: "list", Text: "Milk", Department: "other", CreatedAt: time.Now().Unix(),
	}))

	req := httptest.NewRequest(http.MethodDelete, "/lists/list/items/milk", nil)
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})

	response := httptest.NewRecorder()

	f.handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)

	var list listapi.List
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &list))
	require.Empty(t, list.Items)
}

func TestDeleteCheckedItems(t *testing.T) {
	f := newFixture(t)
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))

	for _, item := range []struct{ id, text string }{{"milk", "Milk"}, {"bread", "Bread"}} {
		require.NoError(t, q.AddItem(t.Context(), listdb.AddItemParams{
			ID:         item.id,
			ListID:     "list",
			Text:       item.text,
			Department: "other",
			CreatedAt:  time.Now().Unix(),
		}))
	}

	require.NoError(t, q.UpdateItem(t.Context(), listdb.UpdateItemParams{
		ID: "milk", ListID: "list", SetChecked: 1, Checked: 1, Now: time.Now().Unix(),
	}))

	req := httptest.NewRequest(http.MethodDelete, "/lists/list/checked-items", nil)
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})

	response := httptest.NewRecorder()

	f.handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)

	var list listapi.List
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &list))
	require.Len(t, list.Items, 1)
	require.Equal(t, "bread", list.Items[0].ID)
	require.Equal(t, "Bread", list.Items[0].Text)
	require.Nil(t, list.Items[0].CheckedAt)
}

func TestListSummaries(t *testing.T) {
	f := newFixture(t)
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))

	for _, item := range []struct{ id, text string }{{"milk", "Milk"}, {"bread", "Bread"}} {
		require.NoError(t, q.AddItem(t.Context(), listdb.AddItemParams{
			ID:         item.id,
			ListID:     "list",
			Text:       item.text,
			Department: "other",
			CreatedAt:  time.Now().Unix(),
		}))
	}

	require.NoError(t, q.DeleteItem(t.Context(), listdb.DeleteItemParams{
		ID: "milk", ListID: "list", Now: time.Now().Unix(),
	}))

	req := httptest.NewRequest(http.MethodGet, "/lists", nil)
	req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})

	response := httptest.NewRecorder()

	f.handler.ServeHTTP(response, req)
	require.Equal(t, http.StatusOK, response.Code)

	var summaries []listapi.Summary
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &summaries))
	require.Len(t, summaries, 1)
	require.Equal(t, "list", summaries[0].ID)
	require.Equal(t, "Groceries", summaries[0].Name)
	require.EqualValues(t, 1, summaries[0].ItemCount)
}

func TestItemQuantities(t *testing.T) {
	f := newFixture(t)
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))
	request := func(t *testing.T, method, path, body string, status int) listapi.List {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: "laga_session", Value: f.owner})
		response := httptest.NewRecorder()
		f.handler.ServeHTTP(response, req)
		require.Equal(t, status, response.Code, response.Body.String())
		var result listapi.List
		if status == http.StatusOK {
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		}
		return result
	}
	id := uuid.NewString()
	path := "/lists/list/items/" + id
	initial := request(
		t,
		http.MethodPost,
		"/lists/list/items",
		`{"id":"`+id+`","text":"Milk","amount":"1,5","unit":"l"}`,
		http.StatusOK,
	)
	require.Equal(t, "1.5", *initial.Items[0].Amount)
	require.Equal(t, listapi.UnitLiter, *initial.Items[0].Unit)
	for _, tc := range []struct {
		name, body, amount string
		unit               listapi.Unit
	}{
		{"amount only", `{"amount":"2.25"}`, "2.25", listapi.UnitLiter},
		{"unit only", `{"unit":"dl"}`, "2.25", listapi.UnitDeciliter},
		{"unrelated patch", `{"checked":true}`, "2.25", listapi.UnitDeciliter},
		{"clear amount", `{"amount":null}`, "", listapi.UnitDeciliter},
		{"set amount without unit change", `{"amount":"9007199254740993.123456789"}`, "9007199254740993.123456789", listapi.UnitDeciliter},
		{"clear unit", `{"unit":null}`, "9007199254740993.123456789", ""},
		{"clear both", `{"amount":null,"unit":null}`, "", ""},
		{"unit without amount", `{"unit":"förp"}`, "", listapi.UnitPackage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := request(t, http.MethodPatch, path, tc.body, http.StatusOK)
			item := result.Items[0]
			require.Equal(t, "Milk", item.Text)
			if tc.amount == "" {
				require.Nil(t, item.Amount)
			} else {
				require.Equal(t, tc.amount, *item.Amount)
			}
			if tc.unit == "" {
				require.Nil(t, item.Unit)
			} else {
				require.Equal(t, tc.unit, *item.Unit)
			}
			// A fresh read verifies persisted state rather than just the mutation response.
			require.Equal(
				t,
				result.Items,
				request(t, http.MethodGet, "/lists/list", "", http.StatusOK).Items,
			)
		})
	}
	for _, body := range []string{
		`{"amount":0}`, `{"amount":"0"}`, `{"amount":"-1"}`, `{"amount":"NaN"}`,
		`{"amount":"1e3"}`, `{"amount":"1.2.3"}`, `{"amount":""}`,
		`{"amount":"` + strings.Repeat("1", 101) + `"}`,
		`{"unit":"kilogram"}`, `{"unit":"KG"}`, `{"unit":""}`, `{"unit":1}`,
		`{"amount":"2","unit":"invalid"}`, `{"text":null}`, `{"checked":null}`,
	} {
		t.Run(body, func(t *testing.T) {
			request(t, http.MethodPatch, path, body, http.StatusBadRequest)
			result := request(t, http.MethodGet, "/lists/list", "", http.StatusOK)
			require.Nil(t, result.Items[0].Amount)
			require.Equal(t, listapi.UnitPackage, *result.Items[0].Unit)
			// Creation follows the same quantity validation as patching.
			if strings.Contains(body, `"text"`) || strings.Contains(body, `"checked"`) {
				return
			}
			createBody := `{"id":"` + uuid.NewString() + `","text":"Invalid",` + body[1:]
			request(t, http.MethodPost, "/lists/list/items", createBody, http.StatusBadRequest)
		})
	}
	for _, unit := range []listapi.Unit{listapi.UnitPiece, listapi.UnitPackage, listapi.UnitKilogram, listapi.UnitGram, listapi.UnitLiter, listapi.UnitDeciliter, listapi.UnitCentiliter, listapi.UnitMilliliter} {
		result := request(t, http.MethodPatch, path, `{"unit":"`+string(unit)+`"}`, http.StatusOK)
		require.Equal(t, unit, *result.Items[0].Unit)
	}
}
