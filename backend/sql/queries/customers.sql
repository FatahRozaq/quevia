-- name: CreateCustomer :one
INSERT INTO customers (guid, tenant_id, user_id, full_name, email, phone, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
RETURNING id, guid, tenant_id, full_name, email, phone, created_at;

-- name: GetCustomerByEmail :one
SELECT id, guid, tenant_id, user_id, full_name, email, phone, created_at, updated_at
FROM customers
WHERE tenant_id = $1 AND lower(email) = lower($2) AND deleted_at IS NULL;

-- name: GetCustomerByGUID :one
SELECT id, guid, tenant_id, user_id, full_name, email, phone, created_at, updated_at
FROM customers
WHERE guid = $1 AND deleted_at IS NULL;
