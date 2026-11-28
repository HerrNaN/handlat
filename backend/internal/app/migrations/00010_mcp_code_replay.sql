-- +goose Up
ALTER TABLE mcp_approvals ADD COLUMN refresh_allowed INTEGER NOT NULL DEFAULT 0 CHECK (refresh_allowed IN (0, 1));
ALTER TABLE mcp_codes ADD COLUMN refresh_allowed INTEGER NOT NULL DEFAULT 0 CHECK (refresh_allowed IN (0, 1));
ALTER TABLE mcp_codes ADD COLUMN spent INTEGER NOT NULL DEFAULT 0 CHECK (spent IN (0, 1));

-- +goose Down
DELETE FROM mcp_codes WHERE spent = 1;
ALTER TABLE mcp_codes DROP COLUMN spent;
ALTER TABLE mcp_codes DROP COLUMN refresh_allowed;
ALTER TABLE mcp_approvals DROP COLUMN refresh_allowed;
