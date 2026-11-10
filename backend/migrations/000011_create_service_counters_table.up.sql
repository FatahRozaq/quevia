CREATE TABLE service_counters (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    code                VARCHAR(20) NOT NULL,
    name                VARCHAR(100) NOT NULL,
    display_order       SMALLINT NOT NULL DEFAULT 0,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_service_counters_guid UNIQUE (guid),
    CONSTRAINT ck_service_counters_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_service_counters_code ON service_counters (branch_id, code) WHERE deleted_at IS NULL;
CREATE INDEX ix_service_counters_tenant_id ON service_counters (tenant_id);
CREATE INDEX ix_service_counters_branch_id ON service_counters (branch_id);
CREATE INDEX ix_service_counters_created_by ON service_counters (created_by);
CREATE INDEX ix_service_counters_updated_by ON service_counters (updated_by);
CREATE INDEX ix_service_counters_deleted_by ON service_counters (deleted_by);

ALTER TABLE service_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_counters FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_service_counters ON service_counters
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
