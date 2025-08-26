-- name: GetAdminSetting :one
SELECT key, value, created_at, updated_at 
FROM admin_settings
WHERE key = $1;

-- name: GetAllAdminSettings :many
SELECT key, value, created_at, updated_at 
FROM admin_settings
ORDER BY key;

-- name: UpsertAdminSetting :exec
INSERT INTO admin_settings (key, value, updated_at) 
VALUES ($1, $2, now())
ON CONFLICT (key) 
DO UPDATE SET 
    value = EXCLUDED.value,
    updated_at = now();

-- name: DeleteAdminSetting :exec
DELETE FROM admin_settings 
WHERE key = $1;

-- name: BulkUpsertAdminSettings :exec
WITH setting_updates(key, value) AS (
    SELECT unnest($1::text[]), unnest($2::jsonb[])
)
INSERT INTO admin_settings (key, value, updated_at) 
SELECT key, value, now() FROM setting_updates
ON CONFLICT (key) 
DO UPDATE SET 
    value = EXCLUDED.value,
    updated_at = now();