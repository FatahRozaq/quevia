CREATE TABLE memberships (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    branch_id           BIGINT REFERENCES branches(id) ON DELETE RESTRICT,
    role                VARCHAR(20) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_memberships_guid UNIQUE (guid),
    CONSTRAINT ck_memberships_role CHECK (role IN ('TENANT_ADMIN', 'STAFF')),
    CONSTRAINT ck_memberships_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_memberships_scope ON memberships (user_id, tenant_id, COALESCE(branch_id, 0)) WHERE deleted_at IS NULL;
CREATE INDEX ix_memberships_tenant_id ON memberships (tenant_id);
CREATE INDEX ix_memberships_user_id ON memberships (user_id);
CREATE INDEX ix_memberships_branch_id ON memberships (branch_id);
CREATE INDEX ix_memberships_created_by ON memberships (created_by);
CREATE INDEX ix_memberships_updated_by ON memberships (updated_by);
CREATE INDEX ix_memberships_deleted_by ON memberships (deleted_by);

ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_memberships ON memberships
    USING (NULLIF(current_setting('app.current_tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
