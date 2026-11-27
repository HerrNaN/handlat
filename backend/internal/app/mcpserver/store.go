package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"
)

const (
	approvalLifetime = 10 * time.Minute
	codeLifetime     = 2 * time.Minute
	accessLifetime   = time.Hour
)

type approval struct {
	ID        string
	ClientID  string
	Name      string
	Redirect  string
	State     string
	Challenge string
}

type connection struct {
	ID        string `json:"id"`
	ClientID  string `json:"clientId"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

var errInvalidGrant = errors.New("invalid grant")

func digest(value string) []byte {
	hash := sha256.Sum256([]byte(value))
	return hash[:]
}

func (h *Handler) saveApproval(ctx context.Context, p approval, browser string) error {
	return h.inTransaction(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(
			ctx,
			"DELETE FROM mcp_approvals WHERE expires_at <= unixepoch()",
		); err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx, `
			DELETE FROM mcp_grants WHERE id IN (
				SELECT grant_id FROM mcp_codes WHERE expires_at <= unixepoch()
			)`,
		); err != nil {
			return err
		}

		_, err := tx.ExecContext(
			ctx,
			`
			INSERT INTO mcp_approvals (id, browser_hash, resource, client_id, client_name, redirect_uri, state, challenge, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID,
			digest(browser),
			h.resource(),
			p.ClientID,
			p.Name,
			p.Redirect,
			p.State,
			p.Challenge,
			time.Now().Add(approvalLifetime).Unix(),
		)

		return err
	})
}

func (h *Handler) loadApproval(ctx context.Context, id, browser string) (approval, error) {
	p := approval{ID: id}
	err := h.db.QueryRowContext(
		ctx, `
		SELECT client_id, client_name, redirect_uri, state, challenge FROM mcp_approvals
		WHERE id = ? AND browser_hash = ? AND resource = ? AND expires_at > unixepoch()`,
		id, digest(browser), h.resource(),
	).Scan(&p.ClientID, &p.Name, &p.Redirect, &p.State, &p.Challenge)

	return p, err
}

func consumeApproval(ctx context.Context, tx *sql.Tx, id, browser string) error {
	result, err := tx.ExecContext(
		ctx, `
		DELETE FROM mcp_approvals WHERE id = ? AND browser_hash = ? AND expires_at > unixepoch()`,
		id, digest(browser),
	)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return sql.ErrNoRows
	}

	return err
}

func (h *Handler) approve(
	ctx context.Context,
	p approval,
	browser string,
	user []byte,
) (string, error) {
	code := rand.Text()

	err := h.inTransaction(ctx, func(tx *sql.Tx) error {
		if err := consumeApproval(ctx, tx, p.ID, browser); err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			"DELETE FROM mcp_grants WHERE user_id = ? AND client_id = ?",
			user,
			p.ClientID,
		); err != nil {
			return err
		}

		grant := rand.Text()
		if _, err := tx.ExecContext(
			ctx, `
			INSERT INTO mcp_grants (id, user_id, client_id, client_name, scope, resource, created_at)
			VALUES (?, ?, ?, ?, ?, ?, unixepoch())`, grant, user, p.ClientID, p.Name, permissionScope, h.resource(),
		); err != nil {
			return err
		}

		_, err := tx.ExecContext(
			ctx, `
			INSERT INTO mcp_codes (code_hash, grant_id, redirect_uri, challenge, expires_at)
			VALUES (?, ?, ?, ?, ?)`, digest(code), grant, p.Redirect, p.Challenge,
			time.Now().Add(codeLifetime).Unix(),
		)

		return err
	})

	return code, err
}

func (h *Handler) exchangeCode(
	ctx context.Context,
	client, code, redirect, verifier string,
) (tokenResponse, error) {
	var tokens tokenResponse

	challenge := base64.RawURLEncoding.EncodeToString(digest(verifier))
	err := h.inTransaction(ctx, func(tx *sql.Tx) error {
		var grant string

		err := tx.QueryRowContext(
			ctx, `
			DELETE FROM mcp_codes WHERE code_hash = ? AND redirect_uri = ? AND challenge = ?
				AND expires_at > unixepoch() AND grant_id IN (
					SELECT id FROM mcp_grants WHERE client_id = ? AND scope = ? AND resource = ?
				) RETURNING grant_id`, digest(code), redirect, challenge, client, permissionScope, h.resource(),
		).Scan(&grant)
		if errors.Is(err, sql.ErrNoRows) {
			return errInvalidGrant
		}

		if err != nil {
			return err
		}

		tokens, err = issueTokens(ctx, tx, grant)

		return err
	})

	return tokens, err
}

func (h *Handler) refresh(ctx context.Context, client, refreshToken string) (tokenResponse, error) {
	var (
		tokens   tokenResponse
		replayed bool
	)

	err := h.inTransaction(ctx, func(tx *sql.Tx) error {
		var grant string

		hash := digest(refreshToken)

		err := tx.QueryRowContext(
			ctx, `
			UPDATE mcp_refresh_tokens SET spent = 1 WHERE token_hash = ? AND spent = 0
				AND grant_id IN (SELECT id FROM mcp_grants WHERE client_id = ? AND scope = ? AND resource = ?)
			RETURNING grant_id`, hash, client, permissionScope, h.resource(),
		).Scan(&grant)
		if errors.Is(err, sql.ErrNoRows) {
			result, err := tx.ExecContext(
				ctx, `
				DELETE FROM mcp_grants WHERE client_id = ? AND resource = ? AND id IN (
					SELECT grant_id FROM mcp_refresh_tokens WHERE token_hash = ? AND spent = 1
				)`, client, h.resource(), hash,
			)
			if err != nil {
				return err
			}

			count, err := result.RowsAffected()

			replayed = count != 0
			if err == nil && !replayed {
				return errInvalidGrant
			}

			return err
		}

		if err != nil {
			return err
		}

		tokens, err = issueTokens(ctx, tx, grant)

		return err
	})
	if err == nil && replayed {
		err = errInvalidGrant
	}

	return tokens, err
}

func issueTokens(ctx context.Context, tx *sql.Tx, grant string) (tokenResponse, error) {
	tokens := tokenResponse{
		AccessToken: rand.Text(), RefreshToken: rand.Text(),
		TokenType: "Bearer", ExpiresIn: int64(accessLifetime / time.Second), Scope: permissionScope,
	}

	if _, err := tx.ExecContext(
		ctx,
		"DELETE FROM mcp_access_tokens WHERE expires_at <= unixepoch()",
	); err != nil {
		return tokens, err
	}

	if _, err := tx.ExecContext(
		ctx, `
		INSERT INTO mcp_access_tokens (token_hash, grant_id, expires_at) VALUES (?, ?, ?)`,
		digest(tokens.AccessToken), grant, time.Now().Add(accessLifetime).Unix(),
	); err != nil {
		return tokens, err
	}

	_, err := tx.ExecContext(
		ctx, "INSERT INTO mcp_refresh_tokens (token_hash, grant_id) VALUES (?, ?)",
		digest(tokens.RefreshToken), grant,
	)

	return tokens, err
}

func (h *Handler) tokenUser(ctx context.Context, token string) ([]byte, error) {
	var user []byte

	err := h.db.QueryRowContext(
		ctx, `
		SELECT g.user_id FROM mcp_access_tokens AS t JOIN mcp_grants AS g ON g.id = t.grant_id
		WHERE t.token_hash = ? AND t.expires_at > unixepoch() AND g.scope = ? AND g.resource = ?`,
		digest(token), permissionScope, h.resource(),
	).Scan(&user)

	return user, err
}

func (h *Handler) inTransaction(ctx context.Context, operation func(*sql.Tx) error) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()

	if err := operation(tx); err != nil {
		return err
	}

	return tx.Commit()
}
