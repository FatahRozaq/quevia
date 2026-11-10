-- name: GetStaffProfileByGUID :one
SELECT id, guid, tenant_id, branch_id, user_id, display_name, specialization, status, created_at, updated_at
FROM staff_profiles
WHERE guid = $1 AND deleted_at IS NULL;

-- name: ListStaffByBranchAndService :many
SELECT sp.id, sp.guid, sp.tenant_id, sp.branch_id, sp.user_id, sp.display_name, sp.specialization, sp.status, sp.created_at, sp.updated_at
FROM staff_profiles sp
INNER JOIN staff_services ss ON sp.id = ss.staff_id AND ss.deleted_at IS NULL
WHERE sp.branch_id = $1 AND ss.service_id = $2 AND sp.status = 'ACTIVE' AND sp.deleted_at IS NULL
ORDER BY sp.display_name ASC;
