CREATE TABLE audit_logs (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT REFERENCES branches(id) ON DELETE RESTRICT,
    action              VARCHAR(60) NOT NULL,
    entity_type         VARCHAR(60) NOT NULL,
    entity_guid         UUID NOT NULL,
    before_data         JSONB,
    after_data          JSONB,
    ip_address          INET,
    request_id          VARCHAR(64),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_audit_logs_guid UNIQUE (guid)
);

CREATE INDEX ix_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX ix_audit_logs_entity ON audit_logs (entity_type, entity_guid);
CREATE INDEX ix_audit_logs_branch_id ON audit_logs (branch_id);
CREATE INDEX ix_audit_logs_created_by ON audit_logs (created_by);

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
