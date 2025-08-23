-- name: CreateUser :exec
INSERT INTO users (id, email, auth_provider, auth_provider_id, account_type, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetUserByID :one
SELECT id, email, auth_provider, auth_provider_id, account_type, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, auth_provider, auth_provider_id, account_type, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByAuthProviderID :one
SELECT id, email, auth_provider, auth_provider_id, account_type, created_at, updated_at
FROM users
WHERE auth_provider = $1 AND auth_provider_id = $2;

-- name: UpdateUser :exec
UPDATE users
SET email = $2, auth_provider = $3, auth_provider_id = $4, account_type = $5, updated_at = $6
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;  