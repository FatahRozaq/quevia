CREATE TABLE customers (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    user_id             BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    full_name           VARCHAR(150) NOT NULL,
    email               VARCHAR(255),
    phone               VARCHAR(30),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_customers_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_customers_email ON customers (tenant_id, lower(email)) WHERE deleted_at IS NULL AND email IS NOT NULL;
CREATE INDEX ix_customers_tenant_id ON customers (tenant_id);
CREATE INDEX ix_customers_user_id ON customers (user_id);
CREATE INDEX ix_customers_created_by ON customers (created_by);
CREATE INDEX ix_customers_updated_by ON customers (updated_by);
CREATE INDEX ix_customers_deleted_by ON customers (deleted_by);

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE customers FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_customers ON customers
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
