-- +goose Up
CREATE TABLE mcp_approvals (
    id TEXT PRIMARY KEY NOT NULL,
    browser_hash BLOB NOT NULL,
    resource TEXT NOT NULL,
    client_id TEXT NOT NULL,
    client_name TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    state TEXT NOT NULL,
    challenge TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX mcp_approvals_expiry ON mcp_approvals (expires_at);

CREATE TABLE mcp_grants (
    id TEXT PRIMARY KEY NOT NULL,
    user_id BLOB NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL,
    client_name TEXT NOT NULL,
    scope TEXT NOT NULL,
    resource TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE (user_id, client_id)
);

CREATE TABLE mcp_codes (
    code_hash BLOB PRIMARY KEY NOT NULL,
    grant_id TEXT NOT NULL UNIQUE REFERENCES mcp_grants(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    challenge TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE TABLE mcp_access_tokens (
    token_hash BLOB PRIMARY KEY NOT NULL,
    grant_id TEXT NOT NULL REFERENCES mcp_grants(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL
);
CREATE INDEX mcp_access_tokens_grant ON mcp_access_tokens (grant_id);
CREATE INDEX mcp_access_tokens_expiry ON mcp_access_tokens (expires_at);

CREATE TABLE mcp_refresh_tokens (
    token_hash BLOB PRIMARY KEY NOT NULL,
    grant_id TEXT NOT NULL REFERENCES mcp_grants(id) ON DELETE CASCADE,
    spent INTEGER NOT NULL DEFAULT 0 CHECK (spent IN (0, 1))
);
CREATE INDEX mcp_refresh_tokens_grant ON mcp_refresh_tokens (grant_id);

-- +goose Down
DROP TABLE mcp_refresh_tokens;
DROP TABLE mcp_access_tokens;
DROP TABLE mcp_codes;
DROP TABLE mcp_grants;
DROP TABLE mcp_approvals;
