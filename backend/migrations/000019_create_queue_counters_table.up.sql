CREATE TABLE queue_counters (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    service_id          BIGINT NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
    queue_date          DATE NOT NULL,
    last_number         INTEGER NOT NULL DEFAULT 0,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_queue_counters_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_queue_counters_scope ON queue_counters (branch_id, service_id, queue_date) WHERE deleted_at IS NULL;
CREATE INDEX ix_queue_counters_tenant_id ON queue_counters (tenant_id);
CREATE INDEX ix_queue_counters_created_by ON queue_counters (created_by);
CREATE INDEX ix_queue_counters_updated_by ON queue_counters (updated_by);
CREATE INDEX ix_queue_counters_deleted_by ON queue_counters (deleted_by);

ALTER TABLE queue_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE queue_counters FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_queue_counters ON queue_counters
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
