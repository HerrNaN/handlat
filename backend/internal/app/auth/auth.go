package auth

import (
	"context"
	"crypto/rand"
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
	"github.com/herrnan/laga/internal/app/auth/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	registrationCookie = "laga_registration"
	loginCookie        = "laga_login"
	sessionCookie      = "laga_session"
	registrationTTL    = 5 * time.Minute
	loginTTL           = 5 * time.Minute
	sessionTTL         = 30 * 24 * time.Hour
	rollbackTimeout    = 5 * time.Second
)

type Handler struct {
	pool    *pgxpool.Pool
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

func New(pool *pgxpool.Pool, rpID, origin string, log *slog.Logger) (*Handler, error) {
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
		RPDisplayName: "Laga",
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("configure passkeys: %w", err)
	}

	return &Handler{pool: pool, queries: db.New(pool), wa: wa, log: log}, nil
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register/begin", h.beginRegistration)
	mux.HandleFunc("POST /api/auth/register/finish", h.finishRegistration)
	mux.HandleFunc("POST /api/auth/login/begin", h.beginLogin)
	mux.HandleFunc("POST /api/auth/login/finish", h.finishLogin)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
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

	u := user{id: id, name: "Laga " + base64.RawURLEncoding.EncodeToString(id[:6])}

	options, ceremony, err := h.wa.BeginRegistration(u)
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
		Ceremony:  ceremonyJSON,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(registrationTTL),
			Valid: true,
		},
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
		http.Error(w, "Registration expired; try again", http.StatusBadRequest)
		return
	}

	hash, err := tokenHash(cookie.Value)
	if err != nil {
		http.Error(w, "Registration expired; try again", http.StatusBadRequest)
		return
	}

	attempt, err := h.queries.PopRegistrationAttempt(r.Context(), hash)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Registration expired; try again", http.StatusBadRequest)
		return
	} else if err != nil {
		h.fail(w, "consume registration attempt", err)
		return
	}

	u := user{id: attempt.UserID, name: attempt.Name}

	var ceremony webauthn.SessionData

	err = json.Unmarshal(attempt.Ceremony, &ceremony)
	if err != nil {
		h.fail(w, "decode registration", err)
		return
	}

	credential, err := h.wa.FinishRegistration(u, ceremony, r)
	if err != nil {
		h.log.Error("Passkey verification failed", "err", err)
		http.Error(w, "Passkey verification failed; try again", http.StatusBadRequest)
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

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		h.fail(w, "begin account transaction", err)
		return
	}
	defer h.rollback(r.Context(), tx)

	queries := h.queries.WithTx(tx)
	err = queries.InsertUser(r.Context(), db.InsertUserParams{ID: u.id, Name: u.name})
	if err != nil {
		h.fail(w, "create account", err)
		return
	}

	err = queries.InsertCredential(r.Context(), db.InsertCredentialParams{
		ID:     credential.ID,
		UserID: u.id,
		Data:   credentialJSON,
	})
	if err != nil {
		h.fail(w, "save registration credential", err)
		return
	}

	err = queries.InsertSession(r.Context(), db.InsertSessionParams{
		TokenHash: sessionHash,
		UserID:    u.id,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(sessionTTL),
			Valid: true,
		},
	})
	if err != nil {
		h.fail(w, "create registration session", err)
		return
	}

	err = tx.Commit(r.Context())
	if err != nil {
		h.fail(w, "commit account transaction", err)
		return
	}

	h.cookie(w, registrationCookie, "", "/api/auth/register", -time.Second)
	h.cookie(w, sessionCookie, sessionToken, "/", sessionTTL)
	h.writeAccount(w, http.StatusCreated, u.id, u.name)
}

func (h *Handler) beginLogin(w http.ResponseWriter, r *http.Request) {
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
		TokenHash: hash,
		Ceremony:  ceremonyJSON,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(loginTTL),
			Valid: true,
		},
	})
	if err != nil {
		h.fail(w, "save login", err)
		return
	}

	h.cookie(w, loginCookie, loginToken, "/api/auth/login", loginTTL)
	h.writeJSON(w, http.StatusOK, options.Response, "write login options")
}

func (h *Handler) finishLogin(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(loginCookie)
	if err != nil {
		http.Error(w, "Sign-in expired; try again", http.StatusBadRequest)
		return
	}

	hash, err := tokenHash(cookie.Value)
	if err != nil {
		http.Error(w, "Sign-in expired; try again", http.StatusBadRequest)
		return
	}

	// Consume the challenge atomically to prevent replaying a login ceremony.
	ceremonyJSON, err := h.queries.PopLoginAttempt(r.Context(), hash)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Sign-in expired; try again", http.StatusBadRequest)
		return
	} else if err != nil {
		h.fail(w, "consume login attempt", err)
		return
	}
	h.cookie(w, loginCookie, "", "/api/auth/login", -time.Second)

	var ceremony webauthn.SessionData
	if err = json.Unmarshal(ceremonyJSON, &ceremony); err != nil {
		h.fail(w, "decode login", err)
		return
	}

	lookup := func(rawID, userHandle []byte) (webauthn.User, error) {
		account, err := h.queries.GetUserByCredential(r.Context(), db.GetUserByCredentialParams{
			CredentialID: rawID,
			UserID:       userHandle,
		})
		if err != nil {
			return nil, err
		}

		var credential webauthn.Credential
		if err = json.Unmarshal(account.Data, &credential); err != nil {
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
		http.Error(w, "Passkey verification failed; try again", http.StatusBadRequest)
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

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		h.fail(w, "begin login transaction", err)
		return
	}
	defer h.rollback(r.Context(), tx)

	queries := h.queries.WithTx(tx)
	err = queries.UpdateCredential(r.Context(), db.UpdateCredentialParams{
		ID: credential.ID, UserID: u.WebAuthnID(), Data: credentialJSON,
	})
	if err != nil {
		h.fail(w, "update credential", err)
		return
	}

	err = queries.InsertSession(r.Context(), db.InsertSessionParams{
		TokenHash: sessionHash,
		UserID:    u.WebAuthnID(),
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(sessionTTL),
			Valid: true,
		},
	})
	if err != nil {
		h.fail(w, "create session", err)
		return
	}

	if err = tx.Commit(r.Context()); err != nil {
		h.fail(w, "complete login", err)
		return
	}

	h.cookie(w, sessionCookie, sessionToken, "/", sessionTTL)
	h.writeAccount(w, http.StatusOK, u.WebAuthnID(), u.WebAuthnName())
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	h.cookie(w, sessionCookie, "", "/", -1)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		http.Error(w, "Not signed in", http.StatusUnauthorized)
		return
	}

	hash, err := tokenHash(cookie.Value)
	if err != nil {
		http.Error(w, "Not signed in", http.StatusUnauthorized)
		return
	}

	account, err := h.queries.GetUserBySession(r.Context(), hash)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Not signed in", http.StatusUnauthorized)
		return
	} else if err != nil {
		h.fail(w, "load session", err)
		return
	}

	h.writeAccount(w, http.StatusOK, account.ID, account.Name)
}

func (h *Handler) rollback(ctx context.Context, tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		h.log.Error("roll back auth transaction", "error", err)
	}
}

func (h *Handler) writeAccount(w http.ResponseWriter, status int, id []byte, name string) {
	// User IDs are binary. Base64 makes them JSON strings; this is not URL escaping.
	h.writeJSON(w, status, map[string]string{
		"id":   base64.RawURLEncoding.EncodeToString(id),
		"name": name,
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
	http.Error(w, "Server error", http.StatusInternalServerError)
}
