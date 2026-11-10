-- name: IncrementQueueCounter :one
INSERT INTO queue_counters (tenant_id, branch_id, service_id, queue_date, last_number, created_by, updated_by)
VALUES ($1, $2, $3, $4, 1, $5, $5)
ON CONFLICT (branch_id, service_id, queue_date) WHERE deleted_at IS NULL
DO UPDATE SET last_number = queue_counters.last_number + 1,
              updated_at = NOW(),
              updated_by = EXCLUDED.updated_by
RETURNING last_number;

-- name: CallNextTicket :one
SELECT id, guid, tenant_id, branch_id, service_id, customer_id, appointment_id, counter_id, staff_id,
       queue_date, queue_number, queue_code, source, priority, status, checked_in_at, created_at
FROM queue_tickets
WHERE branch_id = $1 AND queue_date = $2 AND status = 'WAITING'
  AND service_id = ANY($3::bigint[]) AND deleted_at IS NULL
ORDER BY priority DESC, COALESCE(checked_in_at, created_at), queue_number
LIMIT 1
FOR UPDATE SKIP LOCKED;
