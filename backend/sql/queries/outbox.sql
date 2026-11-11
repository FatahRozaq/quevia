-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (guid, tenant_id, aggregate_type, aggregate_guid, event_type, payload, status, available_at, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, 'PENDING', NOW(), $7, $7)
RETURNING id, guid, created_at;
