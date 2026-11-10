CREATE TABLE idempotency_keys (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    scope               VARCHAR(60) NOT NULL,
    key                 VARCHAR(100) NOT NULL,
    request_hash        VARCHAR(64) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'IN_PROGRESS',
    response_status     SMALLINT,
    response_body       JSONB,
    expires_at          TIMESTAMPTZ NOT NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_idempotency_keys_guid UNIQUE (guid),
    CONSTRAINT ck_idempotency_keys_status CHECK (status IN ('IN_PROGRESS', 'COMPLETED'))
);

CREATE UNIQUE INDEX uq_idempotency_keys_scope ON idempotency_keys (tenant_id, scope, key);
CREATE INDEX ix_idempotency_keys_expires ON idempotency_keys (expires_at);
CREATE INDEX ix_idempotency_keys_tenant_id ON idempotency_keys (tenant_id);
CREATE INDEX ix_idempotency_keys_created_by ON idempotency_keys (created_by);
CREATE INDEX ix_idempotency_keys_updated_by ON idempotency_keys (updated_by);

ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE idempotency_keys FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_idempotency_keys ON idempotency_keys
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
