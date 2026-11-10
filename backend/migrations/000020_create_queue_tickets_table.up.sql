CREATE TABLE queue_tickets (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    branch_id           BIGINT NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    service_id          BIGINT NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
    customer_id         BIGINT REFERENCES customers(id) ON DELETE RESTRICT,
    appointment_id      BIGINT REFERENCES appointments(id) ON DELETE RESTRICT,
    counter_id          BIGINT REFERENCES service_counters(id) ON DELETE RESTRICT,
    staff_id            BIGINT REFERENCES staff_profiles(id) ON DELETE RESTRICT,

    queue_date          DATE NOT NULL,
    queue_number        INTEGER NOT NULL,
    queue_code          VARCHAR(20) NOT NULL,
    source              VARCHAR(20) NOT NULL,
    priority            INTEGER NOT NULL DEFAULT 0,
    status              VARCHAR(30) NOT NULL DEFAULT 'WAITING',

    check_in_method     VARCHAR(20),
    checked_in_at       TIMESTAMPTZ,
    called_at           TIMESTAMPTZ,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    recall_count        SMALLINT NOT NULL DEFAULT 0,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_queue_tickets_guid UNIQUE (guid),
    CONSTRAINT ck_queue_tickets_source CHECK (source IN ('WALK_IN', 'APPOINTMENT')),
    CONSTRAINT ck_queue_tickets_status CHECK (
        status IN ('WAITING', 'CALLED', 'SERVING', 'COMPLETED', 'SKIPPED', 'CANCELLED', 'NO_SHOW')
    ),
    CONSTRAINT ck_queue_tickets_source_appointment CHECK (
        (source = 'APPOINTMENT') = (appointment_id IS NOT NULL)
    ),
    CONSTRAINT ck_queue_tickets_check_in_method CHECK (
        check_in_method IS NULL OR check_in_method IN ('WEB', 'QR', 'BOOKING_CODE', 'OVERRIDE')
    )
);

CREATE UNIQUE INDEX uq_queue_tickets_appointment ON queue_tickets (appointment_id) WHERE appointment_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_queue_tickets_number ON queue_tickets (branch_id, service_id, queue_date, queue_number) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_queue_tickets_code ON queue_tickets (branch_id, queue_date, queue_code) WHERE deleted_at IS NULL;
CREATE INDEX ix_queue_tickets_tenant_id ON queue_tickets (tenant_id);
CREATE INDEX ix_queue_tickets_call_next ON queue_tickets (branch_id, queue_date, status, priority DESC, queue_number);
CREATE INDEX ix_queue_tickets_completed ON queue_tickets (branch_id, queue_date) WHERE status = 'COMPLETED' AND deleted_at IS NULL;
CREATE INDEX ix_queue_tickets_customer_id ON queue_tickets (customer_id);
CREATE INDEX ix_queue_tickets_counter_id ON queue_tickets (counter_id);
CREATE INDEX ix_queue_tickets_staff_id ON queue_tickets (staff_id);
CREATE INDEX ix_queue_tickets_created_by ON queue_tickets (created_by);
CREATE INDEX ix_queue_tickets_updated_by ON queue_tickets (updated_by);
CREATE INDEX ix_queue_tickets_deleted_by ON queue_tickets (deleted_by);

ALTER TABLE queue_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE queue_tickets FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_queue_tickets ON queue_tickets
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
