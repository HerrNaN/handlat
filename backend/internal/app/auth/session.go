package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/herrnan/handlat/internal/app/auth/db"
	"github.com/herrnan/handlat/internal/util"
)

type accountKey struct{}

// UserID is set by the authentication middleware.
func UserID(ctx context.Context) []byte {
	id, _ := ctx.Value(accountKey{}).([]byte)
	return id
}

// WithUserID attaches an already authenticated account to an internal request.
func WithUserID(ctx context.Context, id []byte) context.Context {
	return context.WithValue(ctx, accountKey{}, id)
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
		account, err := h.sessionAccount(r)
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("Cache-Control", "no-store")
			util.WriteAPIError(w, http.StatusUnauthorized, "not_signed_in", nil)

			return
		}

		if err != nil {
			w.Header().Set("Cache-Control", "no-store")
			h.fail(w, "load session", err)

			return
		}

		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), account.ID)))
	})
}
