package mcpserver

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/herrnan/handlat/internal/app/auth"
	"github.com/herrnan/handlat/internal/util"
)

const (
	permissionScope = "handlat:lists-recipes"
	approvalCookie  = "handlat_mcp_approval"
	resourcePath    = "/.well-known/oauth-protected-resource/mcp"
	issuerPath      = "/.well-known/oauth-authorization-server"
)

type Handler struct {
	db       *sql.DB
	log      *slog.Logger
	auth     *auth.Handler
	origin   string
	enabled  bool
	metadata *http.Client
}

func New(
	db *sql.DB,
	log *slog.Logger,
	authHandler *auth.Handler,
	origin string,
	enabled bool,
) *Handler {
	return &Handler{
		db:       db,
		log:      log,
		auth:     authHandler,
		origin:   origin,
		enabled:  enabled,
		metadata: metadataHTTPClient(),
	}
}

func (h *Handler) Register(mux, apiMux *http.ServeMux, operations http.Handler, version string) {
	apiMux.Handle("GET /connected-apps", h.auth.RequireSession(http.HandlerFunc(h.connections)))
	apiMux.Handle(
		"DELETE /connected-apps/{id}",
		h.auth.RequireSession(http.HandlerFunc(h.disconnect)),
	)

	if !h.enabled {
		mux.Handle("/mcp", http.NotFoundHandler())
		mux.Handle("/oauth/", http.NotFoundHandler())
		mux.Handle(resourcePath, http.NotFoundHandler())
		mux.Handle(issuerPath, http.NotFoundHandler())

		return
	}

	mux.HandleFunc("GET "+resourcePath, h.resourceMetadata)
	mux.HandleFunc("GET "+issuerPath, h.issuerMetadata)
	mux.HandleFunc("GET /oauth/authorize", h.authorize)
	mux.HandleFunc("POST /oauth/token", h.token)
	mux.Handle("/mcp", h.requireToken(newToolHandler(operations, h.origin, version)))

	apiMux.HandleFunc("GET /mcp/approvals/{id}", h.approvalInfo)
	apiMux.HandleFunc("POST /mcp/approvals/{id}/passkey/begin", h.beginPasskey)
	apiMux.HandleFunc("POST /mcp/approvals/{id}/passkey/finish", h.finishPasskey)
	apiMux.HandleFunc("POST /mcp/approvals/{id}/deny", h.deny)
}

func (h *Handler) resource() string {
	return h.origin + "/mcp"
}

func (h *Handler) resourceMetadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":              h.resource(),
		"resource_name":         "Handlat",
		"authorization_servers": []string{h.origin},
		"scopes_supported": []string{
			permissionScope,
		},
		"bearer_methods_supported": []string{"header"},
	})
}

func (h *Handler) issuerMetadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                   h.origin,
		"authorization_endpoint":   h.origin + "/oauth/authorize",
		"token_endpoint":           h.origin + "/oauth/token",
		"response_types_supported": []string{"code"},
		"grant_types_supported": []string{
			"authorization_code",
			"refresh_token",
		},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"scopes_supported":                               []string{permissionScope},
		"authorization_response_iss_parameter_supported": true,
		"client_id_metadata_document_supported":          true,
	})
}

var pkceVerifier = regexp.MustCompile(`^[a-zA-Z0-9._~-]{43,128}$`)

func validChallenge(challenge string) bool {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(challenge)
	return err == nil && len(decoded) == 32
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")

	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(r.URL.RawQuery) > 8192 || hasDuplicates(query) {
		oauthError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	p := approval{
		ID: rand.Text(), ClientID: query.Get("client_id"), Redirect: query.Get("redirect_uri"),
		State: query.Get("state"), Challenge: query.Get("code_challenge"),
	}
	if len(p.State) > 2048 {
		oauthError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	problem := ""

	switch {
	case query.Get("response_type") != "code":
		problem = "unsupported_response_type"
	case query.Get("resource") != h.resource():
		problem = "invalid_target"
	case query.Get("scope") != "" && query.Get("scope") != permissionScope:
		problem = "invalid_scope"
	case query.Get("code_challenge_method") != "S256" || !validChallenge(p.Challenge):
		problem = "invalid_request"
	}

	if problem != "" {
		// The browser has yet to approve navigation to this client's callback.
		oauthError(w, http.StatusBadRequest, problem)
		return
	}

	client, err := readClient(r.Context(), h.metadata, p.ClientID)
	if err != nil ||
		!slices.ContainsFunc(
			client.Redirects,
			func(uri string) bool { return redirectMatches(uri, p.Redirect) },
		) {
		oauthError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	if !client.supportsCode() {
		oauthError(w, http.StatusBadRequest, "unauthorized_client")
		return
	}

	p.Name = client.Name
	p.RefreshAllowed = slices.Contains(client.GrantTypes, "refresh_token")

	browser := rand.Text()
	if err := h.saveApproval(r.Context(), p, browser); err != nil {
		h.fail(w, err)
		return
	}

	setApprovalCookie(w, p.ID, browser, int(approvalLifetime.Seconds()))
	http.Redirect(w, r, "/connect?request="+p.ID, http.StatusSeeOther)
}

func (h *Handler) callback(p approval, key, value string) string {
	target, _ := url.Parse(p.Redirect) // Validated against the client metadata.
	query := target.Query()
	query.Set(key, value)
	query.Set("iss", h.origin)

	if p.State != "" {
		query.Set("state", p.State)
	}

	target.RawQuery = query.Encode()

	return target.String()
}

func (h *Handler) token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Pragma", "no-cache")
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err != nil || contentType != "application/x-www-form-urlencoded" ||
		r.ParseForm() != nil || hasDuplicates(r.PostForm) ||
		r.Header.Get("Authorization") != "" || r.PostForm.Has("client_secret") {
		oauthError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	data := r.PostForm

	client := data.Get("client_id")
	if _, ok := httpsClientID(client); !ok {
		oauthError(w, http.StatusBadRequest, "invalid_client")
		return
	}

	grantType, resource := data.Get("grant_type"), data.Get("resource")
	if resource != h.resource() && (resource != "" || grantType == "authorization_code") {
		oauthError(w, http.StatusBadRequest, "invalid_target")
		return
	}

	if scope := data.Get("scope"); scope != "" && scope != permissionScope {
		oauthError(w, http.StatusBadRequest, "invalid_scope")
		return
	}

	var tokens tokenResponse

	switch grantType {
	case "authorization_code":
		verifier := data.Get("code_verifier")
		if !pkceVerifier.MatchString(verifier) || data.Get("code") == "" ||
			data.Get("redirect_uri") == "" {
			oauthError(w, http.StatusBadRequest, "invalid_request")
			return
		}

		tokens, err = h.exchangeCode(
			r.Context(),
			client,
			data.Get("code"),
			data.Get("redirect_uri"),
			verifier,
		)
	case "refresh_token":
		if data.Get("refresh_token") == "" {
			oauthError(w, http.StatusBadRequest, "invalid_request")
			return
		}

		tokens, err = h.refresh(r.Context(), client, data.Get("refresh_token"))
	default:
		oauthError(w, http.StatusBadRequest, "unsupported_grant_type")
		return
	}

	if errors.Is(err, errInvalidGrant) {
		oauthError(w, http.StatusBadRequest, "invalid_grant")
		return
	}

	if err != nil {
		h.fail(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

func (h *Handler) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		if origin := r.Header.Get("Origin"); origin != "" && origin != h.origin {
			http.Error(w, "Origin is not allowed", http.StatusForbidden)
			return
		}

		header := r.Header.Values("Authorization")
		if len(header) == 1 {
			scheme, token, _ := strings.Cut(header[0], " ")
			if strings.EqualFold(scheme, "Bearer") && token != "" && len(token) <= 256 {
				user, err := h.tokenUser(r.Context(), token)
				if err == nil {
					next.ServeHTTP(w, r.WithContext(auth.WithUserID(r.Context(), user)))
					return
				}

				if !errors.Is(err, sql.ErrNoRows) {
					h.fail(w, err)
					return
				}
			}
		}

		challenge := `Bearer resource_metadata="` + h.origin + resourcePath + `", scope="` + permissionScope + `"`
		if len(header) != 0 {
			challenge += `, error="invalid_token"`
		}

		w.Header().Set("WWW-Authenticate", challenge)
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func hasDuplicates(values url.Values) bool {
	for _, value := range values {
		if len(value) != 1 {
			return true
		}
	}

	return false
}

func oauthError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	h.log.Error("MCP request failed", "error", err)
	w.Header().Set("Cache-Control", "no-store")
	util.WriteAPIError(w, http.StatusInternalServerError, "internal_error", nil)
}
