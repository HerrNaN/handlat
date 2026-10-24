-- name: InsertRegistrationAttempt :exec
INSERT INTO registration_attempts (token_hash, user_id, name, ceremony, expires_at)
VALUES (@token_hash, @user_id, @name, @ceremony, @expires_at);

-- name: PopRegistrationAttempt :one
DELETE FROM registration_attempts
WHERE token_hash = @token_hash AND expires_at > unixepoch()
RETURNING user_id, name, ceremony;

-- name: InsertUser :exec
INSERT INTO users (id, name)
VALUES (@id, @name);

-- name: InsertCredential :exec
INSERT INTO credentials (id, user_id, data)
VALUES (@id, @user_id, @data);

-- name: InsertSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at)
VALUES (@token_hash, @user_id, @expires_at);

-- name: GetUserBySession :one
SELECT users.id, users.name
FROM sessions
JOIN users ON users.id = sessions.user_id
WHERE sessions.token_hash = @token_hash AND sessions.expires_at > unixepoch();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = @token_hash;

-- name: InsertLoginAttempt :exec
INSERT INTO login_attempts (token_hash, ceremony, expires_at)
VALUES (@token_hash, @ceremony, @expires_at);

-- name: PopLoginAttempt :one
DELETE FROM login_attempts
WHERE token_hash = @token_hash AND expires_at > unixepoch()
RETURNING ceremony;

-- name: GetUserByCredential :one
SELECT users.id, users.name, credentials.data
FROM credentials
JOIN users ON users.id = credentials.user_id
WHERE credentials.id = @credential_id AND users.id = @user_id;

-- name: UpdateCredential :exec
UPDATE credentials SET data = @data
WHERE id = @id AND user_id = @user_id;
