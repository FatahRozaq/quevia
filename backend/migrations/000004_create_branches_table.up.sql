CREATE TABLE branches (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    code                VARCHAR(20) NOT NULL,
    name                VARCHAR(150) NOT NULL,
    address             TEXT,
    phone               VARCHAR(30),
    timezone            VARCHAR(50) NOT NULL DEFAULT 'Asia/Jakarta',
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_branches_guid UNIQUE (guid),
    CONSTRAINT ck_branches_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_branches_tenant_code ON branches (tenant_id, code) WHERE deleted_at IS NULL;
CREATE INDEX ix_branches_tenant_id ON branches (tenant_id);
CREATE INDEX ix_branches_created_by ON branches (created_by);
CREATE INDEX ix_branches_updated_by ON branches (updated_by);
CREATE INDEX ix_branches_deleted_by ON branches (deleted_by);

ALTER TABLE branches ENABLE ROW LEVEL SECURITY;
ALTER TABLE branches FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_branches ON branches
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
