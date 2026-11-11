-- name: GetIdempotencyKey :one
SELECT id, guid, tenant_id, scope, key, request_hash, status, response_status, response_body, expires_at, created_at, updated_at
FROM idempotency_keys
WHERE tenant_id = $1 AND scope = $2 AND key = $3;

-- name: CreateIdempotencyKey :one
INSERT INTO idempotency_keys (guid, tenant_id, scope, key, request_hash, status, expires_at, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
RETURNING id, guid, tenant_id, scope, key, request_hash, status, expires_at, created_at;

-- name: CompleteIdempotencyKey :exec
UPDATE idempotency_keys
SET status = 'COMPLETED', response_status = $4, response_body = $5, updated_at = NOW(), updated_by = $6
WHERE tenant_id = $1 AND scope = $2 AND key = $3;
