-- name: CreateMembership :one
INSERT INTO memberships (guid, tenant_id, user_id, branch_id, role, status, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
RETURNING id, guid, tenant_id, user_id, branch_id, role, status, created_at;

-- name: GetMembershipsByUserID :many
SELECT m.id, m.guid, m.tenant_id, m.user_id, m.branch_id, m.role, m.status,
       t.guid as tenant_guid, t.name as tenant_name, t.slug as tenant_slug,
       b.guid as branch_guid, b.name as branch_name, b.code as branch_code
FROM memberships m
INNER JOIN tenants t ON m.tenant_id = t.id AND t.deleted_at IS NULL
LEFT JOIN branches b ON m.branch_id = b.id AND b.deleted_at IS NULL
WHERE m.user_id = $1 AND m.status = 'ACTIVE' AND m.deleted_at IS NULL;
