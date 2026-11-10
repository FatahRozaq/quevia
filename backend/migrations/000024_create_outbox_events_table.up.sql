CREATE TABLE outbox_events (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    guid                UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id           BIGINT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    aggregate_type      VARCHAR(60) NOT NULL,
    aggregate_guid      UUID NOT NULL,
    event_type          VARCHAR(80) NOT NULL,
    payload             JSONB NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    attempts            SMALLINT NOT NULL DEFAULT 0,
    available_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at        TIMESTAMPTZ,
    last_error          TEXT,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          BIGINT REFERENCES users(id) ON DELETE RESTRICT,

    CONSTRAINT uq_outbox_events_guid UNIQUE (guid),
    CONSTRAINT ck_outbox_events_status CHECK (status IN ('PENDING', 'PROCESSED', 'FAILED'))
);

CREATE INDEX ix_outbox_events_worker_pending ON outbox_events (status, available_at) WHERE status = 'PENDING';
CREATE INDEX ix_outbox_events_tenant_id ON outbox_events (tenant_id);
CREATE INDEX ix_outbox_events_created_by ON outbox_events (created_by);
CREATE INDEX ix_outbox_events_updated_by ON outbox_events (updated_by);

ALTER TABLE outbox_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox_events FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_outbox_events ON outbox_events
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::BIGINT);
