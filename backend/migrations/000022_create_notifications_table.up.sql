CREATE TABLE notifications (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    appointment_id      BIGINT REFERENCES appointments(id) ON DELETE RESTRICT,
    queue_ticket_id     BIGINT REFERENCES queue_tickets(id) ON DELETE RESTRICT,
    customer_id         BIGINT REFERENCES customers(id) ON DELETE RESTRICT,
    recipient_email     VARCHAR(255),
    event_type          VARCHAR(40) NOT NULL,
    channel             VARCHAR(10) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    subject             VARCHAR(255),
    body                TEXT NOT NULL,
    scheduled_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at             TIMESTAMPTZ,
    attempts            SMALLINT NOT NULL DEFAULT 0,
    last_error          TEXT,
    read_at             TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at          TIMESTAMPTZ,
    deleted_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_notifications_guid UNIQUE (guid),
    CONSTRAINT ck_notifications_channel CHECK (channel IN ('EMAIL', 'IN_APP')),
    CONSTRAINT ck_notifications_status CHECK (status IN ('PENDING', 'SENT', 'FAILED', 'CANCELLED')),
    CONSTRAINT ck_notifications_email_recipient CHECK (channel != 'EMAIL' OR recipient_email IS NOT NULL)
);

CREATE INDEX ix_notifications_tenant_id ON notifications (tenant_id);
CREATE INDEX ix_notifications_worker_pending ON notifications (status, scheduled_at) WHERE status = 'PENDING';
CREATE INDEX ix_notifications_appointment_id ON notifications (appointment_id);
CREATE INDEX ix_notifications_queue_ticket_id ON notifications (queue_ticket_id);
CREATE INDEX ix_notifications_customer_id ON notifications (customer_id);
CREATE INDEX ix_notifications_created_by ON notifications (created_by);
CREATE INDEX ix_notifications_updated_by ON notifications (updated_by);
CREATE INDEX ix_notifications_deleted_by ON notifications (deleted_by);

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_notifications ON notifications
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
