CREATE TABLE service_categories (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    name                VARCHAR(100) NOT NULL,
    sort_order          SMALLINT NOT NULL DEFAULT 0,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_service_categories_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_service_categories_name ON service_categories (tenant_id, lower(name)) WHERE deleted_at IS NULL;
CREATE INDEX ix_service_categories_tenant_id ON service_categories (tenant_id);
CREATE INDEX ix_service_categories_created_by ON service_categories (created_by);
CREATE INDEX ix_service_categories_updated_by ON service_categories (updated_by);
CREATE INDEX ix_service_categories_deleted_by ON service_categories (deleted_by);

ALTER TABLE service_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_categories FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_service_categories ON service_categories
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
