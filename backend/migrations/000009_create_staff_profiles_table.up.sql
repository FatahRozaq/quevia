CREATE TABLE staff_profiles (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    user_id             BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    display_name        VARCHAR(150) NOT NULL,
    specialization      VARCHAR(100),
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_staff_profiles_guid UNIQUE (guid),
    CONSTRAINT ck_staff_profiles_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE UNIQUE INDEX uq_staff_profiles_user_branch ON staff_profiles (user_id, branch_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_staff_profiles_tenant_id ON staff_profiles (tenant_id);
CREATE INDEX ix_staff_profiles_branch_id ON staff_profiles (branch_id);
CREATE INDEX ix_staff_profiles_user_id ON staff_profiles (user_id);
CREATE INDEX ix_staff_profiles_created_by ON staff_profiles (created_by);
CREATE INDEX ix_staff_profiles_updated_by ON staff_profiles (updated_by);
CREATE INDEX ix_staff_profiles_deleted_by ON staff_profiles (deleted_by);

ALTER TABLE staff_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE staff_profiles FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_staff_profiles ON staff_profiles
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
