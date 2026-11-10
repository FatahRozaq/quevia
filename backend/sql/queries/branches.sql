-- name: GetBranchByGUID :one
SELECT id, guid, tenant_id, code, name, address, phone, timezone, status, created_at, updated_at
FROM branches
WHERE guid = $1 AND deleted_at IS NULL;

-- name: ListBranchesByTenant :many
SELECT id, guid, tenant_id, code, name, address, phone, timezone, status, created_at, updated_at
FROM branches
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY code ASC;

-- name: CreateBranch :one
INSERT INTO branches (guid, tenant_id, code, name, address, phone, timezone, status, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
RETURNING id, guid, tenant_id, code, name, address, phone, timezone, status, created_at;
