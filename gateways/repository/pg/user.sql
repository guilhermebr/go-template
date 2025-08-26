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

-- name: ListUsers :many
SELECT id, email, auth_provider, auth_provider_id, account_type, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountUsersByAccountType :one
SELECT COUNT(*) FROM users WHERE account_type = $1;

-- name: GetUserStats :one
SELECT 
    COUNT(*) as total_users,
    COUNT(CASE WHEN account_type = 'admin' THEN 1 END) as admin_users,
    COUNT(CASE WHEN account_type = 'super_admin' THEN 1 END) as super_admin_users,
    COUNT(CASE WHEN account_type = 'user' THEN 1 END) as regular_users,
    COUNT(CASE WHEN created_at >= NOW() - INTERVAL '7 days' THEN 1 END) as recent_signups
FROM users;  