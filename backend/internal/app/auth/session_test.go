package auth

import (
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/herrnan/handlat/internal/app/auth/db"
	"github.com/herrnan/handlat/internal/app/database"
	"github.com/herrnan/handlat/internal/app/migrations"
	"github.com/stretchr/testify/require"
)

func TestRequireSession(t *testing.T) {
	conn, err := database.Open(t.Context(), ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, migrations.Up(conn))
	q := db.New(conn)
	require.NoError(t, q.InsertUser(t.Context(), db.InsertUserParams{ID: []byte("user"), Name: "User"}))
	token, hash, err := newToken()
	require.NoError(t, err)
	require.NoError(t, q.InsertSession(t.Context(), db.InsertSessionParams{TokenHash: hash, UserID: []byte("user"), ExpiresAt: time.Now().Add(time.Hour).Unix()}))
	h := &Handler{queries: q, log: slog.Default()}
	protected := h.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, []byte("user"), UserID(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, tc := range []struct {
		name, token string
		status      int
	}{
		{"missing", "", 401}, {"malformed", "bad", 401}, {"valid", token, 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.token != "" {
				req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tc.token})
			}
			res := httptest.NewRecorder()
			protected.ServeHTTP(res, req)
			require.Equal(t, tc.status, res.Code)
			require.Equal(t, "no-store", res.Header().Get("Cache-Control"))
		})
	}
	logoutRequest := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	logoutResponse := httptest.NewRecorder()
	h.logout(logoutResponse, logoutRequest)
	require.Equal(t, 200, logoutResponse.Code)
	_, err = q.GetUserBySession(t.Context(), hash)
	require.ErrorIs(t, err, sql.ErrNoRows, "logout must revoke the server session")
	require.NoError(t, q.InsertSession(t.Context(), db.InsertSessionParams{TokenHash: hash, UserID: []byte("user"), ExpiresAt: time.Now().Add(time.Hour).Unix()}))
	_, err = conn.ExecContext(t.Context(), "UPDATE sessions SET expires_at = 0")
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	res := httptest.NewRecorder()
	protected.ServeHTTP(res, req)
	require.Equal(t, 401, res.Code)
}
