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

-- name: GetBranchSettings :one
SELECT id, guid, tenant_id, branch_id, change_cutoff_minutes, check_in_open_before_minutes,
       check_in_close_after_minutes, no_show_after_minutes, slot_step_minutes,
       appointment_priority, walk_in_priority, late_grace_minutes, notify_before_queue_count,
       created_at, updated_at
FROM branch_settings
WHERE branch_id = $1 AND deleted_at IS NULL;

-- name: UpsertBranchSettings :one
INSERT INTO branch_settings (
    guid, tenant_id, branch_id, change_cutoff_minutes, check_in_open_before_minutes,
    check_in_close_after_minutes, no_show_after_minutes, slot_step_minutes,
    appointment_priority, walk_in_priority, late_grace_minutes, notify_before_queue_count,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    $9, $10, $11, $12,
    $13, $13
)
ON CONFLICT (branch_id) WHERE deleted_at IS NULL
DO UPDATE SET
    change_cutoff_minutes = EXCLUDED.change_cutoff_minutes,
    check_in_open_before_minutes = EXCLUDED.check_in_open_before_minutes,
    check_in_close_after_minutes = EXCLUDED.check_in_close_after_minutes,
    no_show_after_minutes = EXCLUDED.no_show_after_minutes,
    slot_step_minutes = EXCLUDED.slot_step_minutes,
    appointment_priority = EXCLUDED.appointment_priority,
    walk_in_priority = EXCLUDED.walk_in_priority,
    late_grace_minutes = EXCLUDED.late_grace_minutes,
    notify_before_queue_count = EXCLUDED.notify_before_queue_count,
    updated_at = NOW(),
    updated_by = EXCLUDED.updated_by
RETURNING id, guid, tenant_id, branch_id, change_cutoff_minutes, check_in_open_before_minutes,
          check_in_close_after_minutes, no_show_after_minutes, slot_step_minutes,
          appointment_priority, walk_in_priority, late_grace_minutes, notify_before_queue_count,
          created_at, updated_at;

-- name: ListBusinessHoursByBranch :many
SELECT id, guid, tenant_id, branch_id, day_of_week, open_time, close_time, created_at, updated_at
FROM business_hours
WHERE branch_id = $1 AND deleted_at IS NULL
ORDER BY day_of_week ASC, open_time ASC;

-- name: DeleteBusinessHoursByBranch :exec
UPDATE business_hours
SET deleted_at = NOW(), deleted_by = $2
WHERE branch_id = $1 AND deleted_at IS NULL;

-- name: CreateBusinessHour :one
INSERT INTO business_hours (guid, tenant_id, branch_id, day_of_week, open_time, close_time, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
RETURNING id, guid, tenant_id, branch_id, day_of_week, open_time, close_time, created_at;

-- name: ListCountersByBranch :many
SELECT id, guid, tenant_id, branch_id, code, name, display_order, status, created_at, updated_at
FROM service_counters
WHERE branch_id = $1 AND deleted_at IS NULL
ORDER BY display_order ASC, code ASC;

-- name: CreateCounter :one
INSERT INTO service_counters (guid, tenant_id, branch_id, code, name, display_order, status, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
RETURNING id, guid, tenant_id, branch_id, code, name, display_order, status, created_at;

-- name: GetCounterByGUID :one
SELECT id, guid, tenant_id, branch_id, code, name, display_order, status, created_at, updated_at
FROM service_counters
WHERE guid = $1 AND deleted_at IS NULL;

-- name: UpdateCounter :one
UPDATE service_counters
SET name = $2, display_order = $3, status = $4, updated_at = NOW(), updated_by = $5
WHERE guid = $1 AND deleted_at IS NULL
RETURNING id, guid, tenant_id, branch_id, code, name, display_order, status, updated_at;
