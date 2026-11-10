CREATE TABLE invitations (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT REFERENCES branches(id) ON DELETE RESTRICT,
    email               VARCHAR(255) NOT NULL,
    role                VARCHAR(20) NOT NULL,
    token_hash          VARCHAR(128) NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    accepted_at         TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_invitations_guid UNIQUE (guid),
    CONSTRAINT ck_invitations_role CHECK (role IN ('TENANT_ADMIN', 'STAFF')),
    CONSTRAINT ck_invitations_status CHECK (status IN ('PENDING', 'ACCEPTED', 'EXPIRED', 'REVOKED'))
);

CREATE UNIQUE INDEX uq_invitations_token_hash ON invitations (token_hash) WHERE deleted_at IS NULL;
CREATE INDEX ix_invitations_tenant_id ON invitations (tenant_id);
CREATE INDEX ix_invitations_branch_id ON invitations (branch_id);
CREATE INDEX ix_invitations_created_by ON invitations (created_by);
CREATE INDEX ix_invitations_updated_by ON invitations (updated_by);
CREATE INDEX ix_invitations_deleted_by ON invitations (deleted_by);

ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE invitations FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_invitations ON invitations
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
