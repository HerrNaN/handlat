package mcpserver

import (
	"database/sql"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

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
	callback, _ := url.Parse(p.Redirect)
	host := strings.ToLower(strings.TrimSuffix(callback.Hostname(), "."))
	ip, _ := netip.ParseAddr(host)
	writeJSON(w, http.StatusOK, map[string]any{
		"name":         p.Name,
		"domain":       client.Host,
		"callbackHost": callback.Host,
		"localCallback": ip.Unmap().IsLoopback() || host == "localhost" ||
			strings.HasSuffix(host, ".localhost"),
		"renewable": p.RefreshAllowed,
	})
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
		SELECT g.id, g.client_id, g.client_name, g.created_at FROM mcp_grants AS g
		WHERE g.user_id = ? AND (
			EXISTS (SELECT 1 FROM mcp_codes WHERE grant_id = g.id AND spent = 0 AND expires_at > unixepoch())
			OR EXISTS (SELECT 1 FROM mcp_access_tokens WHERE grant_id = g.id AND expires_at > unixepoch())
			OR EXISTS (SELECT 1 FROM mcp_refresh_tokens WHERE grant_id = g.id AND spent = 0)
		) ORDER BY g.created_at DESC, g.id`, auth.UserID(r.Context()),
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
