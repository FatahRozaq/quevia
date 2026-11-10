-- name: GetServiceByGUID :one
SELECT id, guid, tenant_id, branch_id, category_id, name, description, duration_minutes, buffer_minutes, price, queue_prefix, required_specialization, status, created_at, updated_at
FROM services
WHERE guid = $1 AND deleted_at IS NULL;

-- name: ListServicesByBranch :many
SELECT id, guid, tenant_id, branch_id, category_id, name, description, duration_minutes, buffer_minutes, price, queue_prefix, required_specialization, status, created_at, updated_at
FROM services
WHERE branch_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL
ORDER BY name ASC;
