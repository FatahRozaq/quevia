CREATE TABLE display_screens (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    name                VARCHAR(100) NOT NULL,
    access_token_hash   VARCHAR(128) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    last_seen_at        TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_display_screens_guid UNIQUE (guid),
    CONSTRAINT ck_display_screens_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE INDEX ix_display_screens_tenant_id ON display_screens (tenant_id);
CREATE INDEX ix_display_screens_branch_id ON display_screens (branch_id);
CREATE INDEX ix_display_screens_access_token ON display_screens (access_token_hash);
CREATE INDEX ix_display_screens_created_by ON display_screens (created_by);
CREATE INDEX ix_display_screens_updated_by ON display_screens (updated_by);
CREATE INDEX ix_display_screens_deleted_by ON display_screens (deleted_by);

ALTER TABLE display_screens ENABLE ROW LEVEL SECURITY;
ALTER TABLE display_screens FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_display_screens ON display_screens
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
