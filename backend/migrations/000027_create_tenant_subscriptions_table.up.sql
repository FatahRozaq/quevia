CREATE TABLE tenant_subscriptions (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    plan_id             BIGINT NOT NULL REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    status              VARCHAR(20) NOT NULL DEFAULT 'TRIAL',
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at             TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_tenant_subscriptions_guid UNIQUE (guid),
    CONSTRAINT ck_tenant_subscriptions_status CHECK (status IN ('TRIAL', 'ACTIVE', 'CANCELLED', 'EXPIRED'))
);

CREATE UNIQUE INDEX uq_tenant_subscriptions_active ON tenant_subscriptions (tenant_id) WHERE status IN ('TRIAL', 'ACTIVE') AND deleted_at IS NULL;
CREATE INDEX ix_tenant_subscriptions_tenant_id ON tenant_subscriptions (tenant_id);
CREATE INDEX ix_tenant_subscriptions_plan_id ON tenant_subscriptions (plan_id);
CREATE INDEX ix_tenant_subscriptions_created_by ON tenant_subscriptions (created_by);
CREATE INDEX ix_tenant_subscriptions_updated_by ON tenant_subscriptions (updated_by);
CREATE INDEX ix_tenant_subscriptions_deleted_by ON tenant_subscriptions (deleted_by);

ALTER TABLE tenant_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_subscriptions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_tenant_subscriptions ON tenant_subscriptions
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
