package list

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/herrnan/handlat/internal/app/auth"
	"github.com/herrnan/handlat/internal/app/list/db"
)

func invitationHash(token string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != 32 {
		return nil, sql.ErrNoRows
	}

	hash := sha256.Sum256(raw)

	return hash[:], nil
}

func (h *Handler) createInvitation(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	tx, err := h.conn.BeginTx(r.Context(), nil)
	if err != nil {
		h.fail(w, err)
		return
	}
	defer h.rollback(tx)

	q := h.queries.WithTx(tx)

	value, err := accessibleList(r.Context(), q, r.PathValue("id"), auth.UserID(r.Context()))
	if err != nil {
		h.fail(w, err)
		return
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		h.fail(w, err)
		return
	}

	hash := sha256.Sum256(raw)
	expiresAt := time.Now().Add(24 * time.Hour)

	if err := q.DeleteExpiredInvitations(r.Context(), now()); err != nil {
		h.fail(w, err)
		return
	}

	if err := q.CreateInvitation(r.Context(), db.CreateInvitationParams{
		TokenHash: hash[:], ListID: value.ID, ExpiresAt: expiresAt.Unix(),
	}); err != nil {
		h.fail(w, err)
		return
	}

	if err := tx.Commit(); err != nil {
		h.fail(w, err)
		return
	}

	h.respond(w, http.StatusCreated, struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expiresAt"`
	}{base64.RawURLEncoding.EncodeToString(raw), expiresAt})
}

func (h *Handler) invitationError(w http.ResponseWriter, err error) {
	w.Header().Set("Cache-Control", "no-store")

	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "This invitation is no longer available.", http.StatusNotFound)
		return
	}

	h.fail(w, err)
}

// Only the list name is exposed by a valid token. Reading never consumes it.
func (h *Handler) getInvitation(w http.ResponseWriter, r *http.Request) {
	hash, err := invitationHash(r.PathValue("token"))
	if err != nil {
		h.invitationError(w, err)
		return
	}

	value, err := h.queries.GetInvitation(
		r.Context(),
		db.GetInvitationParams{TokenHash: hash, Now: now()},
	)
	if err != nil {
		h.invitationError(w, err)
		return
	}

	h.respond(w, http.StatusOK, struct {
		Name string `json:"name"`
	}{value.Name})
}

func (h *Handler) join(r *http.Request) (List, error) {
	hash, err := invitationHash(r.PathValue("token"))
	if err != nil {
		return List{}, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	tx, err := h.conn.BeginTx(r.Context(), nil)
	if err != nil {
		return List{}, err
	}
	defer h.rollback(tx)

	q := h.queries.WithTx(tx)

	id, err := q.ConsumeInvitation(
		r.Context(),
		db.ConsumeInvitationParams{TokenHash: hash, Now: now()},
	)
	if err != nil {
		return List{}, err
	}

	if err := q.AddMember(r.Context(), db.AddMemberParams{ListID: id, UserID: auth.UserID(r.Context())}); err != nil {
		return List{}, err
	}

	value, err := q.GetList(r.Context(), id)
	if err != nil {
		return List{}, err
	}

	result, err := snapshot(r.Context(), q, value)
	if err != nil {
		return List{}, err
	}

	if err := tx.Commit(); err != nil {
		return List{}, err
	}

	h.publish(id, listEvent{List: result})

	return result, nil
}

func (h *Handler) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	result, err := h.join(r)
	if err != nil {
		h.invitationError(w, err)
		return
	}

	h.respond(w, http.StatusOK, result)
}
