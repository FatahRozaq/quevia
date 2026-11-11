-- name: CreateAuditLog :one
INSERT INTO audit_logs (guid, tenant_id, branch_id, action, entity_type, entity_guid, before_data, after_data, ip_address, request_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, guid, created_at;
