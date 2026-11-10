CREATE TABLE schedule_exceptions (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    staff_id            BIGINT REFERENCES staff_profiles(id) ON DELETE RESTRICT,
    exception_type      VARCHAR(20) NOT NULL,
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    start_time          TIME,
    end_time            TIME,
    reason              VARCHAR(255),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_schedule_exceptions_guid UNIQUE (guid),
    CONSTRAINT ck_schedule_exceptions_type CHECK (exception_type IN ('HOLIDAY', 'LEAVE', 'CUSTOM_HOURS')),
    CONSTRAINT ck_schedule_exceptions_dates CHECK (end_date >= start_date),
    CONSTRAINT ck_schedule_exceptions_times CHECK (
        (start_time IS NULL AND end_time IS NULL) OR
        (start_time IS NOT NULL AND end_time IS NOT NULL AND end_time > start_time)
    )
);

CREATE INDEX ix_schedule_exceptions_tenant_id ON schedule_exceptions (tenant_id);
CREATE INDEX ix_schedule_exceptions_branch_dates ON schedule_exceptions (branch_id, start_date, end_date);
CREATE INDEX ix_schedule_exceptions_staff_id ON schedule_exceptions (staff_id);
CREATE INDEX ix_schedule_exceptions_created_by ON schedule_exceptions (created_by);
CREATE INDEX ix_schedule_exceptions_updated_by ON schedule_exceptions (updated_by);
CREATE INDEX ix_schedule_exceptions_deleted_by ON schedule_exceptions (deleted_by);

ALTER TABLE schedule_exceptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE schedule_exceptions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_schedule_exceptions ON schedule_exceptions
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
