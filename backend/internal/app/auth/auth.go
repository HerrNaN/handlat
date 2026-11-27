package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/herrnan/handlat/internal/app/auth/db"
	"github.com/herrnan/handlat/internal/util"
)

const (
	registrationCookie = "laga_registration"
	loginCookie        = "laga_login"
	sessionCookie      = "laga_session"
	registrationTTL    = 5 * time.Minute
	loginTTL           = 5 * time.Minute
	sessionTTL         = 30 * 24 * time.Hour
)

type Handler struct {
	dbConn  *sql.DB
	queries *db.Queries
	wa      *webauthn.WebAuthn
	log     *slog.Logger
}

type user struct {
	id          []byte
	name        string
	credentials []webauthn.Credential
}

var _ webauthn.User = user{}

func (u user) WebAuthnID() []byte                         { return u.id }
func (u user) WebAuthnName() string                       { return u.name }
func (u user) WebAuthnDisplayName() string                { return u.name }
func (u user) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func New(conn *sql.DB, rpID, origin string, log *slog.Logger) (*Handler, error) {
	if rpID == "" || origin == "" {
		return nil, errors.New("rp id and origin are required")
	}

	originURL, err := url.Parse(origin)
	if err != nil || originURL.Host == "" || originURL.User != nil || originURL.Path != "" ||
		originURL.RawQuery != "" || originURL.Fragment != "" {
		return nil, errors.New(
			"origin must be an absolute URL without credentials, path, query, or fragment",
		)
	}

	if originURL.Scheme != "https" &&
		(originURL.Scheme != "http" || originURL.Hostname() != "localhost") {
		return nil, errors.New("origin must use HTTPS except for localhost development")
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPOrigins:     []string{origin},
		RPDisplayName: "Handlat",
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("configure passkeys: %w", err)
	}

	return &Handler{dbConn: conn, queries: db.New(conn), wa: wa, log: log}, nil
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register/begin", h.beginRegistration)
	mux.HandleFunc("POST /auth/register/finish", h.finishRegistration)
	mux.HandleFunc("POST /auth/login/begin", h.beginLogin)
	mux.HandleFunc("POST /auth/login/finish", h.finishLogin)
	mux.HandleFunc("POST /auth/logout", h.logout)
	mux.HandleFunc("GET /auth/me", h.me)
}

func (h *Handler) cookie(w http.ResponseWriter, name, value, path string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
	id := make([]byte, 32)
	if _, err := rand.Read(id); err != nil {
		h.fail(w, "generate user ID", err)
		return
	}

	name, err := generateUsername()
	if err != nil {
		h.fail(w, "generate username", err)
		return
	}

	u := user{id: id, name: name}

	options, ceremony, err := h.wa.BeginRegistration(
		u,
		webauthn.WithExtensions(webauthn.WithExtensionCredProps()),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
	)
	if err != nil {
		h.fail(w, "begin registration", err)
		return
	}

	registrationToken, hash, err := newToken()
	if err != nil {
		h.fail(w, "generate registration token", err)
		return
	}

	ceremonyJSON, err := json.Marshal(ceremony)
	if err != nil {
		h.fail(w, "encode registration", err)
		return
	}

	err = h.queries.InsertRegistrationAttempt(r.Context(), db.InsertRegistrationAttemptParams{
		TokenHash: hash,
		UserID:    u.id,
		Name:      u.name,
		Ceremony:  string(ceremonyJSON),
		ExpiresAt: time.Now().Add(registrationTTL).Unix(),
	})
	if err != nil {
		h.fail(w, "save registration", err)
		return
	}

	h.cookie(w, registrationCookie, registrationToken, "/api/auth/register", registrationTTL)
	h.writeJSON(w, http.StatusOK, options.Response, "write registration options")
}

func (h *Handler) finishRegistration(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(registrationCookie)
	if err != nil {
		util.WriteAPIError(w, http.StatusBadRequest, "registration_expired", nil)
		return
	}

	hash, err := tokenHash(cookie.Value)
	if err != nil {
		util.WriteAPIError(w, http.StatusBadRequest, "registration_expired", nil)
		return
	}

	attempt, err := h.queries.PopRegistrationAttempt(r.Context(), hash)
	if errors.Is(err, sql.ErrNoRows) {
		util.WriteAPIError(w, http.StatusBadRequest, "registration_expired", nil)
		return
	} else if err != nil {
		h.fail(w, "consume registration attempt", err)
		return
	}

	u := user{id: attempt.UserID, name: attempt.Name}

	var ceremony webauthn.SessionData

	err = json.Unmarshal([]byte(attempt.Ceremony), &ceremony)
	if err != nil {
		h.fail(w, "decode registration", err)
		return
	}

	credential, err := h.wa.FinishRegistration(u, ceremony, r)
	if err != nil {
		h.log.Error("Passkey verification failed", "err", err)
		util.WriteAPIError(w, http.StatusBadRequest, "passkey_verification_failed", nil)

		return
	}

	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		h.fail(w, "encode credential", err)
		return
	}

	sessionToken, sessionHash, err := newToken()
	if err != nil {
		h.fail(w, "generate session token", err)
		return
	}

	tx, err := h.dbConn.BeginTx(r.Context(), nil)
	if err != nil {
		h.fail(w, "begin account transaction", err)
		return
	}
	defer h.rollback(tx)

	queries := h.queries.WithTx(tx)

	err = queries.InsertUser(r.Context(), db.InsertUserParams{ID: u.id, Name: u.name})
	if err != nil {
		h.fail(w, "create account", err)
		return
	}

	err = queries.InsertCredential(r.Context(), db.InsertCredentialParams{
		ID:     credential.ID,
		UserID: u.id,
		Data:   string(credentialJSON),
	})
	if err != nil {
		h.fail(w, "save registration credential", err)
		return
	}

	err = queries.InsertSession(r.Context(), db.InsertSessionParams{
		TokenHash: sessionHash,
		UserID:    u.id,
		ExpiresAt: time.Now().Add(sessionTTL).Unix(),
	})
	if err != nil {
		h.fail(w, "create registration session", err)
		return
	}

	err = tx.Commit()
	if err != nil {
		h.fail(w, "commit account transaction", err)
		return
	}

	h.cookie(w, registrationCookie, "", "/api/auth/register", -time.Second)
	h.cookie(w, sessionCookie, sessionToken, "/", sessionTTL)
	h.writeAccount(w, http.StatusCreated, u.id, u.name, false)
}

func passkeyTokenHash(hash []byte, path string) []byte {
	// Keep existing login challenges valid across deployments.
	if path == "/api/auth/login" {
		return hash
	}

	scoped := sha256.Sum256(append(hash, path...))

	return scoped[:]
}

func (h *Handler) beginLogin(w http.ResponseWriter, r *http.Request) {
	h.BeginPasskey(w, r, "/api/auth/login")
}

// BeginPasskey starts a discoverable passkey ceremony bound to a browser path.
func (h *Handler) BeginPasskey(w http.ResponseWriter, r *http.Request, path string) {
	options, ceremony, err := h.wa.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		h.fail(w, "begin login", err)
		return
	}

	loginToken, hash, err := newToken()
	if err != nil {
		h.fail(w, "generate login token", err)
		return
	}

	ceremonyJSON, err := json.Marshal(ceremony)
	if err != nil {
		h.fail(w, "encode login", err)
		return
	}

	err = h.queries.InsertLoginAttempt(r.Context(), db.InsertLoginAttemptParams{
		TokenHash: passkeyTokenHash(hash, path),
		Ceremony:  string(ceremonyJSON),
		ExpiresAt: time.Now().Add(loginTTL).Unix(),
	})
	if err != nil {
		h.fail(w, "save login", err)
		return
	}

	h.cookie(w, loginCookie, loginToken, path, loginTTL)
	h.writeJSON(w, http.StatusOK, options.Response, "write login options")
}

func (h *Handler) verifyPasskey(
	w http.ResponseWriter,
	r *http.Request,
	path string,
) (webauthn.User, *webauthn.Credential) {
	cookie, err := r.Cookie(loginCookie)
	if err != nil {
		util.WriteAPIError(w, http.StatusBadRequest, "sign_in_expired", nil)
		return nil, nil
	}

	hash, err := tokenHash(cookie.Value)
	if err != nil {
		util.WriteAPIError(w, http.StatusBadRequest, "sign_in_expired", nil)
		return nil, nil
	}

	// Consume the challenge atomically to prevent replaying a login ceremony.
	ceremonyJSON, err := h.queries.PopLoginAttempt(r.Context(), passkeyTokenHash(hash, path))
	if errors.Is(err, sql.ErrNoRows) {
		util.WriteAPIError(w, http.StatusBadRequest, "sign_in_expired", nil)
		return nil, nil
	} else if err != nil {
		h.fail(w, "consume login attempt", err)
		return nil, nil
	}

	h.cookie(w, loginCookie, "", path, -time.Second)

	var ceremony webauthn.SessionData
	if err = json.Unmarshal([]byte(ceremonyJSON), &ceremony); err != nil {
		h.fail(w, "decode login", err)
		return nil, nil
	}

	ctx := r.Context()

	lookup := func(rawID, userHandle []byte) (webauthn.User, error) {
		account, err := h.queries.GetUserByCredential(ctx, db.GetUserByCredentialParams{
			CredentialID: rawID,
			UserID:       userHandle,
		})
		if err != nil {
			return nil, err
		}

		var credential webauthn.Credential
		if err = json.Unmarshal([]byte(account.Data), &credential); err != nil {
			return nil, err
		}

		return user{
			id:          account.ID,
			name:        account.Name,
			credentials: []webauthn.Credential{credential},
		}, nil
	}

	u, credential, err := h.wa.FinishPasskeyLogin(lookup, ceremony, r)
	if err != nil {
		h.log.Error("Passkey verification failed", "err", err)
		util.WriteAPIError(w, http.StatusBadRequest, "passkey_verification_failed", nil)

		return nil, nil
	}

	return u, credential
}

// ConfirmPasskey verifies a fresh ceremony for the signed-in account, or
// identifies the account from the passkey when the browser has no session.
func (h *Handler) ConfirmPasskey(w http.ResponseWriter, r *http.Request, path string) []byte {
	u, credential := h.verifyPasskey(w, r, path)
	if u == nil {
		return nil
	}

	account, err := h.sessionAccount(r)
	if err == nil && !bytes.Equal(account.ID, u.WebAuthnID()) {
		util.WriteAPIError(w, http.StatusForbidden, "passkey_account_mismatch", nil)
		return nil
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.fail(w, "load session for passkey confirmation", err)
		return nil
	}

	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		h.fail(w, "encode confirmed passkey", err)
		return nil
	}

	err = h.queries.UpdateCredential(r.Context(), db.UpdateCredentialParams{
		ID: credential.ID, UserID: u.WebAuthnID(), Data: string(credentialJSON),
	})
	if err != nil {
		h.fail(w, "update confirmed passkey", err)
		return nil
	}

	return u.WebAuthnID()
}

func (h *Handler) finishLogin(w http.ResponseWriter, r *http.Request) {
	u, credential := h.verifyPasskey(w, r, "/api/auth/login")
	if u == nil {
		return
	}

	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		h.fail(w, "encode credential", err)
		return
	}

	sessionToken, sessionHash, err := newToken()
	if err != nil {
		h.fail(w, "generate session token", err)
		return
	}

	tx, err := h.dbConn.BeginTx(r.Context(), nil)
	if err != nil {
		h.fail(w, "begin login transaction", err)
		return
	}
	defer h.rollback(tx)

	queries := h.queries.WithTx(tx)

	err = queries.UpdateCredential(r.Context(), db.UpdateCredentialParams{
		ID: credential.ID, UserID: u.WebAuthnID(), Data: string(credentialJSON),
	})
	if err != nil {
		h.fail(w, "update credential", err)
		return
	}

	err = queries.InsertSession(r.Context(), db.InsertSessionParams{
		TokenHash: sessionHash,
		UserID:    u.WebAuthnID(),
		ExpiresAt: time.Now().Add(sessionTTL).Unix(),
	})
	if err != nil {
		h.fail(w, "create session", err)
		return
	}

	if err = tx.Commit(); err != nil {
		h.fail(w, "complete login", err)
		return
	}

	h.cookie(w, sessionCookie, sessionToken, "/", sessionTTL)
	h.writeAccount(w, http.StatusOK, u.WebAuthnID(), u.WebAuthnName(), false)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if hash, err := tokenHash(cookie.Value); err == nil {
			if err := h.queries.DeleteSession(r.Context(), hash); err != nil {
				h.fail(w, "revoke session", err)
				return
			}
		}
	}

	h.cookie(w, sessionCookie, "", "/", -1)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")

	account, err := h.sessionAccount(r)
	if errors.Is(err, sql.ErrNoRows) {
		util.WriteAPIError(w, http.StatusUnauthorized, "not_signed_in", nil)
		return
	} else if err != nil {
		h.fail(w, "load session", err)
		return
	}

	h.writeAccount(w, http.StatusOK, account.ID, account.Name, account.RecipesEnabled == 1)
}

func (h *Handler) rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		h.log.Error("roll back auth transaction", "error", err)
	}
}

func (h *Handler) writeAccount(
	w http.ResponseWriter,
	status int,
	id []byte,
	name string,
	recipesEnabled bool,
) {
	// User IDs are binary. Base64 makes them JSON strings; this is not URL escaping.
	h.writeJSON(w, status, struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		RecipesEnabled bool   `json:"recipesEnabled,omitempty"`
	}{
		ID:             base64.RawURLEncoding.EncodeToString(id),
		Name:           name,
		RecipesEnabled: recipesEnabled,
	}, "write account response")
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, body any, operation string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Headers may already be sent, so log the failure rather than writing a second response.
		h.log.Error(operation, "error", err)
	}
}

func (h *Handler) fail(w http.ResponseWriter, operation string, err error) {
	h.log.Error(operation, "error", err)
	util.WriteAPIError(w, http.StatusInternalServerError, "server_error", nil)
}
