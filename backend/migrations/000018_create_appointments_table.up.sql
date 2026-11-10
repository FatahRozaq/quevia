CREATE TABLE appointments (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    customer_id         BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    service_id          BIGINT NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
    staff_id            BIGINT NOT NULL REFERENCES staff_profiles(id) ON DELETE RESTRICT,

    booking_code        VARCHAR(30) NOT NULL,
    qr_token            VARCHAR(64) NOT NULL,
    start_at            TIMESTAMPTZ NOT NULL,
    end_at              TIMESTAMPTZ NOT NULL,
    status              VARCHAR(30) NOT NULL DEFAULT 'CONFIRMED',
    notes               TEXT,
    rescheduled_count   SMALLINT NOT NULL DEFAULT 0,
    cancelled_at        TIMESTAMPTZ,
    cancel_reason       VARCHAR(255),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_appointments_guid UNIQUE (guid),
    CONSTRAINT ck_appointments_time CHECK (end_at > start_at),
    CONSTRAINT ck_appointments_status CHECK (
        status IN ('CONFIRMED', 'CHECKED_IN', 'COMPLETED', 'CANCELLED', 'NO_SHOW')
    ),
    CONSTRAINT ex_appointments_no_overlap EXCLUDE USING gist (
        staff_id WITH =,
        tstzrange(start_at, end_at) WITH &&
    ) WHERE (status IN ('CONFIRMED', 'CHECKED_IN') AND deleted_at IS NULL)
);

CREATE UNIQUE INDEX uq_appointments_booking_code ON appointments (booking_code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_appointments_qr_token ON appointments (qr_token) WHERE deleted_at IS NULL;
CREATE INDEX ix_appointments_tenant_id ON appointments (tenant_id);
CREATE INDEX ix_appointments_branch_start ON appointments (branch_id, start_at) WHERE deleted_at IS NULL;
CREATE INDEX ix_appointments_customer_history ON appointments (customer_id, start_at DESC);
CREATE INDEX ix_appointments_status_start ON appointments (start_at) WHERE status = 'CONFIRMED' AND deleted_at IS NULL;
CREATE INDEX ix_appointments_staff_id ON appointments (staff_id);
CREATE INDEX ix_appointments_service_id ON appointments (service_id);
CREATE INDEX ix_appointments_created_by ON appointments (created_by);
CREATE INDEX ix_appointments_updated_by ON appointments (updated_by);
CREATE INDEX ix_appointments_deleted_by ON appointments (deleted_by);

ALTER TABLE appointments ENABLE ROW LEVEL SECURITY;
ALTER TABLE appointments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_appointments ON appointments
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
