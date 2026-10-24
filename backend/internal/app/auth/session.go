package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/herrnan/handlat/internal/app/auth/db"
)

type accountKey struct{}

// UserID is only populated by RequireSession, never from client input.
func UserID(ctx context.Context) []byte {
	account, _ := ctx.Value(accountKey{}).(db.GetUserBySessionRow)
	return account.ID
}

func (h *Handler) sessionAccount(r *http.Request) (db.GetUserBySessionRow, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return db.GetUserBySessionRow{}, sql.ErrNoRows
	}
	hash, err := tokenHash(cookie.Value)
	if err != nil {
		return db.GetUserBySessionRow{}, sql.ErrNoRows
	}
	return h.queries.GetUserBySession(r.Context(), hash)
}

// RequireSession protects APIs and makes the authenticated account available.
func (h *Handler) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		account, err := h.sessionAccount(r)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Not signed in", http.StatusUnauthorized)
			return
		}
		if err != nil {
			h.fail(w, "load session", err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), accountKey{}, account)))
	})
}
