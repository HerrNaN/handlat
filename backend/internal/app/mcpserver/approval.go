package mcpserver

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"

	"github.com/herrnan/handlat/internal/app/auth"
	"github.com/herrnan/handlat/internal/util"
)

func approvalPath(id string) string {
	return "/api/mcp/approvals/" + id
}

func setApprovalCookie(w http.ResponseWriter, id, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: approvalCookie, Value: value, Path: approvalPath(id),
		HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: maxAge,
	})
}

func (h *Handler) browserApproval(w http.ResponseWriter, r *http.Request) (approval, string, bool) {
	w.Header().Set("Cache-Control", "no-store")

	cookie, err := r.Cookie(approvalCookie)
	if err != nil {
		util.WriteAPIError(w, http.StatusNotFound, "mcp_approval_unavailable", nil)
		return approval{}, "", false
	}

	p, err := h.loadApproval(r.Context(), r.PathValue("id"), cookie.Value)
	if errors.Is(err, sql.ErrNoRows) {
		util.WriteAPIError(w, http.StatusNotFound, "mcp_approval_unavailable", nil)
		return approval{}, "", false
	}

	if err != nil {
		h.fail(w, err)
		return approval{}, "", false
	}

	return p, cookie.Value, true
}

func (h *Handler) approvalInfo(w http.ResponseWriter, r *http.Request) {
	p, _, ok := h.browserApproval(w, r)
	if !ok {
		return
	}

	client, _ := url.Parse(p.ClientID)
	writeJSON(w, http.StatusOK, map[string]string{"name": p.Name, "domain": client.Host})
}

func (h *Handler) beginPasskey(w http.ResponseWriter, r *http.Request) {
	if p, _, ok := h.browserApproval(w, r); ok {
		h.auth.BeginPasskey(w, r, approvalPath(p.ID)+"/passkey")
	}
}

func (h *Handler) finishPasskey(w http.ResponseWriter, r *http.Request) {
	p, browser, ok := h.browserApproval(w, r)
	if !ok {
		return
	}

	user := h.auth.ConfirmPasskey(w, r, approvalPath(p.ID)+"/passkey")
	if user == nil {
		return
	}

	code, err := h.approve(r.Context(), p, browser, user)
	if err != nil {
		h.approvalFailure(w, err)
		return
	}

	setApprovalCookie(w, p.ID, "", -1)
	writeJSON(w, http.StatusOK, map[string]string{"redirect": h.callback(p, "code", code)})
}

func (h *Handler) deny(w http.ResponseWriter, r *http.Request) {
	p, browser, ok := h.browserApproval(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	err := h.inTransaction(ctx, func(tx *sql.Tx) error {
		return consumeApproval(ctx, tx, p.ID, browser)
	})
	if err != nil {
		h.approvalFailure(w, err)
		return
	}

	setApprovalCookie(w, p.ID, "", -1)
	writeJSON(
		w,
		http.StatusOK,
		map[string]string{"redirect": h.callback(p, "error", "access_denied")},
	)
}

func (h *Handler) approvalFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		util.WriteAPIError(w, http.StatusNotFound, "mcp_approval_unavailable", nil)
		return
	}

	h.fail(w, err)
}

func (h *Handler) connections(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(
		r.Context(), `
		SELECT id, client_id, client_name, created_at FROM mcp_grants
		WHERE user_id = ? AND NOT EXISTS (
			SELECT 1 FROM mcp_codes WHERE grant_id = mcp_grants.id AND expires_at <= unixepoch()
		) ORDER BY created_at DESC, id`, auth.UserID(r.Context()),
	)
	if err != nil {
		h.fail(w, err)
		return
	}
	defer func() { _ = rows.Close() }()

	apps := []connection{}

	for rows.Next() {
		var c connection
		if err := rows.Scan(&c.ID, &c.ClientID, &c.Name, &c.CreatedAt); err != nil {
			h.fail(w, err)
			return
		}

		apps = append(apps, c)
	}

	if err := rows.Err(); err != nil {
		h.fail(w, err)
		return
	}

	writeJSON(w, http.StatusOK, struct {
		Enabled bool         `json:"enabled"`
		Apps    []connection `json:"apps"`
	}{h.enabled, apps})
}

func (h *Handler) disconnect(w http.ResponseWriter, r *http.Request) {
	_, err := h.db.ExecContext(
		r.Context(), "DELETE FROM mcp_grants WHERE id = ? AND user_id = ?",
		r.PathValue("id"), auth.UserID(r.Context()),
	)
	if err != nil {
		h.fail(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
