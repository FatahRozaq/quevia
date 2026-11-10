CREATE TABLE services (
    id                      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                    UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id               BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id               BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    category_id             BIGINT REFERENCES service_categories(id) ON DELETE RESTRICT,
    name                    VARCHAR(150) NOT NULL,
    description             TEXT,
    duration_minutes        INTEGER NOT NULL,
    buffer_minutes          INTEGER NOT NULL DEFAULT 0,
    price                   NUMERIC(14,2),
    queue_prefix            VARCHAR(5) NOT NULL,
    required_specialization VARCHAR(100),
    status                  VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by              BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at              TIMESTAMPTZ,
    deleted_by              BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_services_guid UNIQUE (guid),
    CONSTRAINT ck_services_duration CHECK (duration_minutes > 0),
    CONSTRAINT ck_services_buffer CHECK (buffer_minutes >= 0),
    CONSTRAINT ck_services_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_services_queue_prefix ON services (branch_id, queue_prefix) WHERE deleted_at IS NULL;
CREATE INDEX ix_services_tenant_id ON services (tenant_id);
CREATE INDEX ix_services_branch_id ON services (branch_id);
CREATE INDEX ix_services_category_id ON services (category_id);
CREATE INDEX ix_services_created_by ON services (created_by);
CREATE INDEX ix_services_updated_by ON services (updated_by);
CREATE INDEX ix_services_deleted_by ON services (deleted_by);

ALTER TABLE services ENABLE ROW LEVEL SECURITY;
ALTER TABLE services FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_services ON services
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
