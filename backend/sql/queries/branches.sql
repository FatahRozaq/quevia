-- name: GetBranchByGUID :one
SELECT id, guid, tenant_id, code, name, address, phone, timezone, status, created_at, updated_at
FROM branches
WHERE guid = $1 AND deleted_at IS NULL;

-- name: ListBranchesByTenant :many
SELECT id, guid, tenant_id, code, name, address, phone, timezone, status, created_at, updated_at
FROM branches
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY code ASC;
