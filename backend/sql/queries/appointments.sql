-- name: GetAppointmentByGUID :one
SELECT id, guid, tenant_id, branch_id, customer_id, service_id, staff_id, booking_code, qr_token, start_at, end_at, status, notes, rescheduled_count, created_at, updated_at
FROM appointments
WHERE guid = $1 AND deleted_at IS NULL;

-- name: CheckStaffOverlap :many
SELECT id, start_at, end_at, status
FROM appointments
WHERE staff_id = @staff_id
  AND deleted_at IS NULL
  AND status IN ('CONFIRMED', 'CHECKED_IN')
  AND tstzrange(start_at, end_at) && tstzrange(@start_at::timestamptz, @end_at::timestamptz);

-- name: ListStaffAppointmentsForDateRange :many
SELECT id, staff_id, start_at, end_at, status
FROM appointments
WHERE staff_id = @staff_id
  AND status IN ('CONFIRMED', 'CHECKED_IN')
  AND deleted_at IS NULL
  AND tstzrange(start_at, end_at) && tstzrange(@start_at::timestamptz, @end_at::timestamptz)
ORDER BY start_at ASC;

-- name: CreateAppointment :one
INSERT INTO appointments (
    guid, tenant_id, branch_id, customer_id, service_id, staff_id,
    booking_code, qr_token, start_at, end_at, status, notes, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12, $13, $13
)
RETURNING id, guid, tenant_id, branch_id, customer_id, service_id, staff_id, booking_code, qr_token, start_at, end_at, status, created_at;

-- name: GetAppointmentByBookingCodeAndEmail :one
SELECT a.id, a.guid, a.tenant_id, a.branch_id, a.customer_id, a.service_id, a.staff_id,
       a.booking_code, a.qr_token, a.start_at, a.end_at, a.status, a.notes, a.rescheduled_count, a.created_at,
       c.full_name as customer_name, c.email as customer_email,
       s.name as service_name, s.duration_minutes,
       sp.guid as staff_guid, sp.display_name as staff_name,
       b.name as branch_name, b.timezone as branch_timezone
FROM appointments a
INNER JOIN customers c ON a.customer_id = c.id AND c.deleted_at IS NULL
INNER JOIN services s ON a.service_id = s.id AND s.deleted_at IS NULL
INNER JOIN staff_profiles sp ON a.staff_id = sp.id AND sp.deleted_at IS NULL
INNER JOIN branches b ON a.branch_id = b.id AND b.deleted_at IS NULL
WHERE a.booking_code = $1 AND lower(c.email) = lower($2) AND a.deleted_at IS NULL;

-- name: RescheduleAppointment :one
UPDATE appointments
SET start_at = $2, end_at = $3, rescheduled_count = rescheduled_count + 1, updated_at = NOW(), updated_by = $4
WHERE guid = $1 AND deleted_at IS NULL
RETURNING id, guid, booking_code, start_at, end_at, status, rescheduled_count, updated_at;

-- name: CancelAppointment :one
UPDATE appointments
SET status = 'CANCELLED', cancelled_at = NOW(), cancel_reason = $2, updated_at = NOW(), updated_by = $3
WHERE guid = $1 AND status IN ('CONFIRMED', 'CHECKED_IN') AND deleted_at IS NULL
RETURNING id, guid, booking_code, status, cancelled_at, cancel_reason, updated_at;
