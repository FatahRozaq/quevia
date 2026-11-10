CREATE TABLE staff_services (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    staff_id            BIGINT NOT NULL REFERENCES staff_profiles(id) ON DELETE RESTRICT,
    service_id          BIGINT NOT NULL REFERENCES services(id) ON DELETE RESTRICT,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_staff_services_guid UNIQUE (guid)
);

CREATE UNIQUE INDEX uq_staff_services_staff_service ON staff_services (staff_id, service_id) WHERE deleted_at IS NULL;
CREATE INDEX ix_staff_services_tenant_id ON staff_services (tenant_id);
CREATE INDEX ix_staff_services_staff_id ON staff_services (staff_id);
CREATE INDEX ix_staff_services_service_id ON staff_services (service_id);
CREATE INDEX ix_staff_services_created_by ON staff_services (created_by);
CREATE INDEX ix_staff_services_updated_by ON staff_services (updated_by);
CREATE INDEX ix_staff_services_deleted_by ON staff_services (deleted_by);

ALTER TABLE staff_services ENABLE ROW LEVEL SECURITY;
ALTER TABLE staff_services FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_staff_services ON staff_services
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
