-- name: GetUserByEmail :one
SELECT id, guid, email, password_hash, full_name, phone, platform_role, status, is_system, email_verified_at, last_login_at, created_at, updated_at
FROM users
WHERE lower(email) = lower($1) AND deleted_at IS NULL;

-- name: GetUserByGUID :one
SELECT id, guid, email, password_hash, full_name, phone, platform_role, status, is_system, email_verified_at, last_login_at, created_at, updated_at
FROM users
WHERE guid = $1 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (guid, email, password_hash, full_name, phone, platform_role, status, is_system)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, guid, email, full_name, phone, platform_role, status, is_system, created_at;
