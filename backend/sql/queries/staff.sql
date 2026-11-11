-- name: GetStaffProfileByGUID :one
SELECT id, guid, tenant_id, branch_id, user_id, display_name, specialization, status, created_at, updated_at
FROM staff_profiles
WHERE guid = $1 AND deleted_at IS NULL;

-- name: ListStaffByBranch :many
SELECT id, guid, tenant_id, branch_id, user_id, display_name, specialization, status, created_at, updated_at
FROM staff_profiles
WHERE branch_id = $1 AND deleted_at IS NULL
ORDER BY display_name ASC;

-- name: ListStaffByBranchAndService :many
SELECT sp.id, sp.guid, sp.tenant_id, sp.branch_id, sp.user_id, sp.display_name, sp.specialization, sp.status, sp.created_at, sp.updated_at
FROM staff_profiles sp
INNER JOIN staff_services ss ON sp.id = ss.staff_id AND ss.deleted_at IS NULL
WHERE sp.branch_id = $1 AND ss.service_id = $2 AND sp.status = 'ACTIVE' AND sp.deleted_at IS NULL
ORDER BY sp.display_name ASC;

-- name: CreateStaffProfile :one
INSERT INTO staff_profiles (guid, tenant_id, branch_id, user_id, display_name, specialization, status, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
RETURNING id, guid, tenant_id, branch_id, user_id, display_name, specialization, status, created_at;

-- name: UpdateStaffProfileStatus :one
UPDATE staff_profiles
SET status = $2, updated_at = NOW(), updated_by = $3
WHERE guid = $1 AND deleted_at IS NULL
RETURNING id, guid, tenant_id, branch_id, user_id, display_name, specialization, status, updated_at;

-- name: CountActiveUpcomingAppointmentsByStaff :one
SELECT COUNT(*)::bigint as active_count
FROM appointments
WHERE staff_id = $1
  AND status = 'CONFIRMED'
  AND end_at > $2
  AND deleted_at IS NULL;

-- name: AssignStaffService :one
INSERT INTO staff_services (guid, tenant_id, staff_id, service_id, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $5)
RETURNING id, guid, tenant_id, staff_id, service_id, created_at;

-- name: RemoveStaffServicesByStaffID :exec
UPDATE staff_services
SET deleted_at = NOW(), deleted_by = $2
WHERE staff_id = $1 AND deleted_at IS NULL;

-- name: ListStaffServices :many
SELECT ss.id, ss.guid, ss.tenant_id, ss.staff_id, ss.service_id,
       s.guid as service_guid, s.name as service_name
FROM staff_services ss
INNER JOIN services s ON ss.service_id = s.id AND s.deleted_at IS NULL
WHERE ss.staff_id = $1 AND ss.deleted_at IS NULL
ORDER BY s.name ASC;

-- name: CreateStaffSchedule :one
INSERT INTO staff_schedules (
    guid, tenant_id, staff_id, day_of_week, start_time, end_time,
    break_start, break_end, valid_from, valid_until, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $11
)
RETURNING id, guid, tenant_id, staff_id, day_of_week, start_time, end_time,
          break_start, break_end, valid_from, valid_until, created_at;

-- name: ListStaffSchedules :many
SELECT id, guid, tenant_id, staff_id, day_of_week, start_time, end_time,
       break_start, break_end, valid_from, valid_until, created_at
FROM staff_schedules
WHERE staff_id = $1 AND deleted_at IS NULL
ORDER BY day_of_week ASC, start_time ASC;

-- name: GetStaffScheduleForDay :one
SELECT id, guid, tenant_id, staff_id, day_of_week, start_time, end_time,
       break_start, break_end, valid_from, valid_until, created_at
FROM staff_schedules
WHERE staff_id = $1 AND day_of_week = $2 AND deleted_at IS NULL
LIMIT 1;

-- name: DeleteStaffSchedulesByStaffID :exec
UPDATE staff_schedules
SET deleted_at = NOW(), deleted_by = $2
WHERE staff_id = $1 AND deleted_at IS NULL;

-- name: CreateScheduleException :one
INSERT INTO schedule_exceptions (
    guid, tenant_id, branch_id, staff_id, exception_type,
    start_date, end_date, start_time, end_time, reason, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10, $11, $11
)
RETURNING id, guid, tenant_id, branch_id, staff_id, exception_type,
          start_date, end_date, start_time, end_time, reason, created_at;

-- name: ListScheduleExceptionsByBranch :many
SELECT id, guid, tenant_id, branch_id, staff_id, exception_type,
       start_date, end_date, start_time, end_time, reason, created_at
FROM schedule_exceptions
WHERE branch_id = $1 AND deleted_at IS NULL
ORDER BY start_date ASC;

-- name: GetScheduleExceptionsForStaffAndDate :many
SELECT id, guid, tenant_id, branch_id, staff_id, exception_type,
       start_date, end_date, start_time, end_time, reason, created_at
FROM schedule_exceptions
WHERE branch_id = $1
  AND (staff_id IS NULL OR staff_id = $2)
  AND $3 BETWEEN start_date AND end_date
  AND deleted_at IS NULL;
