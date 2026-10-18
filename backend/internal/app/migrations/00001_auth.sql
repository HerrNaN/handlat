-- +goose Up
CREATE TABLE users (
    id BLOB PRIMARY KEY NOT NULL,
    name text NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE credentials (
    id BLOB PRIMARY KEY NOT NULL,
    user_id BLOB NOT NULL UNIQUE REFERENCES users(id),
    data TEXT NOT NULL
);

CREATE TABLE registration_attempts (
    token_hash BLOB PRIMARY KEY NOT NULL,
    user_id BLOB NOT NULL,
    name text NOT NULL,
    ceremony TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE TABLE sessions (
    token_hash BLOB PRIMARY KEY NOT NULL,
    user_id BLOB NOT NULL REFERENCES users(id),
    expires_at INTEGER NOT NULL
);

CREATE TABLE login_attempts (
    token_hash BLOB PRIMARY KEY NOT NULL,
    ceremony TEXT NOT NULL,
    expires_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE login_attempts;
DROP TABLE sessions;
DROP TABLE registration_attempts;
DROP TABLE credentials;
DROP TABLE users;
