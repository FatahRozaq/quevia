CREATE TABLE staff_schedules (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    staff_id            BIGINT NOT NULL REFERENCES staff_profiles(id) ON DELETE RESTRICT,
    day_of_week         SMALLINT NOT NULL,
    start_time          TIME NOT NULL,
    end_time            TIME NOT NULL,
    break_start         TIME,
    break_end           TIME,
    valid_from          DATE NOT NULL,
    valid_until         DATE,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_staff_schedules_guid UNIQUE (guid),
    CONSTRAINT ck_staff_schedules_day CHECK (day_of_week BETWEEN 1 AND 7),
    CONSTRAINT ck_staff_schedules_time CHECK (end_time > start_time),
    CONSTRAINT ck_staff_schedules_break CHECK (
        (break_start IS NULL AND break_end IS NULL) OR
        (break_start IS NOT NULL AND break_end IS NOT NULL AND break_end > break_start AND break_start >= start_time AND break_end <= end_time)
    )
);

CREATE INDEX ix_staff_schedules_tenant_id ON staff_schedules (tenant_id);
CREATE INDEX ix_staff_schedules_staff_day ON staff_schedules (staff_id, day_of_week);
CREATE INDEX ix_staff_schedules_created_by ON staff_schedules (created_by);
CREATE INDEX ix_staff_schedules_updated_by ON staff_schedules (updated_by);
CREATE INDEX ix_staff_schedules_deleted_by ON staff_schedules (deleted_by);

ALTER TABLE staff_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE staff_schedules FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_staff_schedules ON staff_schedules
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
