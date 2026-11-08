package list_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authdb "github.com/herrnan/handlat/internal/app/auth/db"
	listapi "github.com/herrnan/handlat/internal/app/list"
	listdb "github.com/herrnan/handlat/internal/app/list/db"
	"github.com/stretchr/testify/require"
)

func (f fixture) user(t *testing.T, name string) string {
	t.Helper()

	q := authdb.New(f.conn)
	require.NoError(
		t,
		q.InsertUser(t.Context(), authdb.InsertUserParams{ID: []byte(name), Name: name}),
	)
	raw := sha256.Sum256([]byte(name))
	hash := sha256.Sum256(raw[:])
	require.NoError(t, q.InsertSession(t.Context(), authdb.InsertSessionParams{
		TokenHash: hash[:], UserID: []byte(name), ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}))

	return base64.RawURLEncoding.EncodeToString(raw[:])
}

func (f fixture) request(method, path, token, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.AddCookie(&http.Cookie{Name: "laga_session", Value: token})
	}

	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, r)

	return response
}

func (f fixture) invite(t *testing.T, token string) string {
	t.Helper()

	response := f.request(http.MethodPost, "/lists/list/invitations", token, "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var invitation struct {
		Token     string
		ExpiresAt time.Time
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &invitation))
	require.WithinDuration(t, time.Now().Add(24*time.Hour), invitation.ExpiresAt, 2*time.Second)

	return "/invitations/" + invitation.Token
}

func TestInvitationLifecycle(t *testing.T) {
	f := newFixture(t)
	other := f.user(t, "other")
	require.NoError(t, seedList(t.Context(), listdb.New(f.conn), seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))
	require.Equal(
		t,
		http.StatusUnauthorized,
		f.request(http.MethodPost, "/lists/list/invitations", "", "").Code,
	)
	require.Equal(
		t,
		http.StatusNotFound,
		f.request(http.MethodPost, "/lists/list/invitations", other, "").Code,
	)

	path := f.invite(t, f.owner)
	for range 2 {
		preview := f.request(http.MethodGet, path, "", "")
		require.Equal(t, http.StatusOK, preview.Code)
		require.JSONEq(t, `{"name":"Groceries"}`, preview.Body.String())
		require.Equal(t, "no-store", preview.Header().Get("Cache-Control"))
	}

	require.Equal(
		t,
		http.StatusUnauthorized,
		f.request(http.MethodPost, path+"/accept", "", "").Code,
	)
	require.Equal(t, http.StatusNotFound, f.request(http.MethodGet, "/lists/list", other, "").Code)
	accepted := f.request(http.MethodPost, path+"/accept", other, "")
	require.Equal(t, http.StatusOK, accepted.Code)

	var list listapi.List
	require.NoError(t, json.Unmarshal(accepted.Body.Bytes(), &list))
	require.EqualValues(t, 2, list.MemberCount)
	require.Equal(
		t,
		http.StatusNotFound,
		f.request(http.MethodPost, path+"/accept", f.owner, "").Code,
	)
	require.Equal(t, http.StatusNotFound, f.request(http.MethodGet, path, "", "").Code)
	require.Equal(
		t,
		http.StatusOK,
		f.request(http.MethodPatch, "/lists/list", other, `{"name":"Weekend"}`).Code,
	)
	summaries := f.request(http.MethodGet, "/lists", other, "")
	require.JSONEq(
		t,
		`[{"id":"list","name":"Weekend","itemCount":0,"memberCount":2}]`,
		summaries.Body.String(),
	)
	// Every member can invite; expiry, invalid tokens and list deletion invalidate links.
	expired := f.invite(t, other)
	_, err := f.conn.ExecContext(
		t.Context(),
		"UPDATE list_invitations SET expires_at = ?",
		time.Now().Unix(),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, f.request(http.MethodGet, expired, "", "").Code)
	require.Equal(
		t,
		http.StatusNotFound,
		f.request(http.MethodPost, expired+"/accept", other, "").Code,
	)
	require.Equal(
		t,
		http.StatusNotFound,
		f.request(http.MethodGet, "/invitations/invalid", "", "").Code,
	)
	pending := f.invite(t, other)
	require.Equal(
		t,
		http.StatusNoContent,
		f.request(http.MethodDelete, "/lists/list", f.owner, "").Code,
	)
	require.Equal(t, http.StatusOK, f.request(http.MethodGet, pending, "", "").Code)
	require.Equal(
		t,
		http.StatusNoContent,
		f.request(http.MethodDelete, "/lists/list", other, "").Code,
	)
	require.Equal(t, http.StatusNotFound, f.request(http.MethodGet, pending, "", "").Code)
}

func TestInvitationAcceptedOnceConcurrently(t *testing.T) {
	f := newFixture(t)
	first, second := f.user(t, "first"), f.user(t, "second")
	require.NoError(t, seedList(t.Context(), listdb.New(f.conn), seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))
	path := f.invite(t, f.owner)
	start := make(chan struct{})
	results := make(chan int, 2)

	for _, token := range []string{first, second} {
		go func() { <-start; results <- f.request(http.MethodPost, path+"/accept", token, "").Code }()
	}

	close(start)
	require.ElementsMatch(t, []int{http.StatusOK, http.StatusNotFound}, []int{<-results, <-results})
	count, err := listdb.New(f.conn).CountMembers(t.Context(), "list")
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
}

func TestLeavingRevokesStreamsAndPreservesOtherMembers(t *testing.T) {
	f := newFixture(t)
	other := f.user(t, "other")
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))
	require.Equal(
		t,
		http.StatusOK,
		f.request(http.MethodPost, f.invite(t, f.owner)+"/accept", other, "").Code,
	)

	server := httptest.NewServer(f.handler)
	defer server.Close()

	departing := openStream(t, server, f.owner, "list")
	defer func() { require.NoError(t, departing.Body.Close()) }()

	remaining := openStream(t, server, other, "list")
	defer func() { require.NoError(t, remaining.Body.Close()) }()

	departingReader, remainingReader := bufio.NewReader(
		departing.Body,
	), bufio.NewReader(
		remaining.Body,
	)
	readSnapshot(t, departingReader)
	readSnapshot(t, remainingReader)
	require.Equal(
		t,
		http.StatusNoContent,
		f.request(http.MethodDelete, "/lists/list", f.owner, "").Code,
	)

	event, err := departingReader.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "event: deleted\n", event)
	require.EqualValues(t, 1, readSnapshot(t, remainingReader).MemberCount)
	require.Equal(
		t,
		http.StatusNotFound,
		f.request(http.MethodPatch, "/lists/list", f.owner, `{"name":"No access"}`).Code,
	)
	require.Equal(
		t,
		http.StatusNotFound,
		f.request(http.MethodPost, "/lists/list/invitations", f.owner, "").Code,
	)
	require.JSONEq(t, `[]`, f.request(http.MethodGet, "/lists", f.owner, "").Body.String())
	require.Equal(
		t,
		http.StatusOK,
		f.request(http.MethodPatch, "/lists/list", other, `{"name":"Still here"}`).Code,
	)
	require.Equal(t, "Still here", readSnapshot(t, remainingReader).Name)
}

func TestLastMembersLeavingConcurrentlyDeletesList(t *testing.T) {
	f := newFixture(t)
	other := f.user(t, "other")
	q := listdb.New(f.conn)
	require.NoError(t, seedList(t.Context(), q, seedListParams{
		ID: "list", UserID: []byte("owner"), Name: "Groceries", Now: time.Now().Unix(),
	}))
	require.NoError(
		t,
		q.AddMember(t.Context(), listdb.AddMemberParams{ListID: "list", UserID: []byte("other")}),
	)
	require.NoError(
		t,
		q.AddItem(
			t.Context(),
			listdb.AddItemParams{
				ID:         "milk",
				ListID:     "list",
				Text:       "Milk",
				Department: "other",
				CreatedAt:  time.Now().Unix(),
			},
		),
	)
	path := f.invite(t, f.owner)
	start, results := make(chan struct{}), make(chan int, 2)

	for _, token := range []string{f.owner, other} {
		go func() { <-start; results <- f.request(http.MethodDelete, "/lists/list", token, "").Code }()
	}

	close(start)
	require.Equal(t, http.StatusNoContent, <-results)
	require.Equal(t, http.StatusNoContent, <-results)
	require.Equal(t, http.StatusNotFound, f.request(http.MethodGet, path, "", "").Code)

	for _, query := range []string{
		"SELECT count(*) FROM lists", "SELECT count(*) FROM items",
		"SELECT count(*) FROM list_members", "SELECT count(*) FROM list_invitations",
	} {
		var count int
		require.NoError(t, f.conn.QueryRowContext(t.Context(), query).Scan(&count))
		require.Zero(t, count, query)
	}
}
