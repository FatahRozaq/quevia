-- name: GetTenantBySlug :one
SELECT id, guid, name, slug, business_type, contact_email, status, created_at, updated_at
FROM tenants
WHERE slug = $1 AND deleted_at IS NULL;

-- name: CreateTenant :one
INSERT INTO tenants (guid, name, slug, business_type, contact_email, status, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
RETURNING id, guid, name, slug, business_type, contact_email, status, created_at;
