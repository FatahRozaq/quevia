CREATE TABLE tenants (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    name                VARCHAR(150) NOT NULL,
    slug                VARCHAR(80) NOT NULL,
    business_type       VARCHAR(30),
    contact_email       VARCHAR(255) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_tenants_guid UNIQUE (guid),
    CONSTRAINT ck_tenants_status CHECK (status IN ('ACTIVE', 'SUSPENDED'))
);

CREATE UNIQUE INDEX uq_tenants_slug ON tenants (slug) WHERE deleted_at IS NULL;
CREATE INDEX ix_tenants_created_by ON tenants (created_by);
CREATE INDEX ix_tenants_updated_by ON tenants (updated_by);
CREATE INDEX ix_tenants_deleted_by ON tenants (deleted_by);

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_tenants ON tenants
    USING (id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
