-- name: GetAppointmentByGUID :one
SELECT id, guid, tenant_id, branch_id, customer_id, service_id, staff_id, booking_code, qr_token, start_at, end_at, status, notes, rescheduled_count, created_at, updated_at
FROM appointments
WHERE guid = $1 AND deleted_at IS NULL;

-- name: CheckStaffOverlap :many
SELECT id, start_at, end_at, status
FROM appointments
WHERE staff_id = $1
  AND deleted_at IS NULL
  AND status IN ('CONFIRMED', 'CHECKED_IN')
  AND tstzrange(start_at, end_at) && tstzrange($2, $3);

-- name: CreateAppointment :one
INSERT INTO appointments (
    guid, tenant_id, branch_id, customer_id, service_id, staff_id,
    booking_code, qr_token, start_at, end_at, status, notes, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11, $12, $13, $13
)
RETURNING id, guid, booking_code, qr_token, start_at, end_at, status, created_at;
