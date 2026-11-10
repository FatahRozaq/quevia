CREATE TABLE business_hours (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    day_of_week         SMALLINT NOT NULL,
    open_time           TIME NOT NULL,
    close_time          TIME NOT NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_business_hours_guid UNIQUE (guid),
    CONSTRAINT ck_business_hours_day CHECK (day_of_week BETWEEN 1 AND 7),
    CONSTRAINT ck_business_hours_time CHECK (close_time > open_time)
);

CREATE UNIQUE INDEX uq_business_hours_scope ON business_hours (branch_id, day_of_week, open_time) WHERE deleted_at IS NULL;
CREATE INDEX ix_business_hours_tenant_id ON business_hours (tenant_id);
CREATE INDEX ix_business_hours_branch_id ON business_hours (branch_id);
CREATE INDEX ix_business_hours_created_by ON business_hours (created_by);
CREATE INDEX ix_business_hours_updated_by ON business_hours (updated_by);
CREATE INDEX ix_business_hours_deleted_by ON business_hours (deleted_by);

ALTER TABLE business_hours ENABLE ROW LEVEL SECURITY;
ALTER TABLE business_hours FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_business_hours ON business_hours
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
