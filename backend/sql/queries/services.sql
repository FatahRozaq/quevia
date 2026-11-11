-- name: GetServiceByGUID :one
SELECT id, guid, tenant_id, branch_id, category_id, name, description, duration_minutes, buffer_minutes, price, queue_prefix, required_specialization, status, created_at, updated_at
FROM services
WHERE guid = $1 AND deleted_at IS NULL;

-- name: ListServicesByBranch :many
SELECT id, guid, tenant_id, branch_id, category_id, name, description, duration_minutes, buffer_minutes, price, queue_prefix, required_specialization, status, created_at, updated_at
FROM services
WHERE branch_id = $1 AND deleted_at IS NULL
ORDER BY name ASC;

-- name: CreateServiceCategory :one
INSERT INTO service_categories (guid, tenant_id, name, sort_order, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $5)
RETURNING id, guid, tenant_id, name, sort_order, created_at;

-- name: ListServiceCategories :many
SELECT id, guid, tenant_id, name, sort_order, created_at, updated_at
FROM service_categories
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY sort_order ASC, name ASC;

-- name: CreateService :one
INSERT INTO services (
    guid, tenant_id, branch_id, category_id, name, description,
    duration_minutes, buffer_minutes, price, queue_prefix, required_specialization, status,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12,
    $13, $13
)
RETURNING id, guid, tenant_id, branch_id, category_id, name, description,
          duration_minutes, buffer_minutes, price, queue_prefix, required_specialization, status, created_at;

-- name: UpdateService :one
UPDATE services
SET category_id = $2, name = $3, description = $4, duration_minutes = $5,
    buffer_minutes = $6, price = $7, queue_prefix = $8, required_specialization = $9,
    status = $10, updated_at = NOW(), updated_by = $11
WHERE guid = $1 AND deleted_at IS NULL
RETURNING id, guid, tenant_id, branch_id, category_id, name, description,
          duration_minutes, buffer_minutes, price, queue_prefix, required_specialization, status, updated_at;

-- name: DeleteService :exec
UPDATE services
SET deleted_at = NOW(), deleted_by = $2
WHERE guid = $1 AND deleted_at IS NULL;
